from __future__ import annotations

import argparse
import json
import re
import sys
import unicodedata
from dataclasses import dataclass, replace
from pathlib import Path
from typing import Any

from .config import load_subflow_config
from .gemini_client import GeminiUnavailableError, create_client, generate_json_response
from .openai_compatible_client import OpenAICompatibleError, generate_json_response as generate_openai_json_response
from .subtitle_io import SubtitleCue, display_width, read_srt, write_srt


MAX_SUBTITLE_LINES = 2
TARGET_LINE_WIDTH = 22
CONTEXT_WINDOW_CUES = 2
IRRELEVANT_FILE_EXTENSIONS = ("srt", "ass", "ssa", "sub", "idx", "sup", "vtt")
KNOWN_NAMED_ENTITY_REPLACEMENTS: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"(?<![A-Za-z])Murph(?![A-Za-z])", re.IGNORECASE), "墨菲"),
    (re.compile(r"(?<![A-Za-z])Tom(?![A-Za-z])", re.IGNORECASE), "汤姆"),
    (re.compile(r"(?<![A-Za-z])Hughie(?![A-Za-z])", re.IGNORECASE), "休伊"),
    (re.compile(r"(?<![A-Za-z])Translucent(?![A-Za-z])", re.IGNORECASE), "透明人"),
    (re.compile(r"(?<![A-Za-z])Homelander(?![A-Za-z])", re.IGNORECASE), "祖国人"),
    (re.compile(r"(?<![A-Za-z])Starlight(?![A-Za-z])", re.IGNORECASE), "星光"),
    (re.compile(r"(?<![A-Za-z])Shockwave(?![A-Za-z])", re.IGNORECASE), "冲击波"),
    (re.compile(r"(?<![A-Za-z])A\s*-\s*Train(?![A-Za-z])", re.IGNORECASE), "火车头"),
    (re.compile(r"(?<![A-Za-z])Mother(?:'s|’s)\s+Milk(?![A-Za-z])", re.IGNORECASE), "母乳"),
    (re.compile(r"(?<![A-Za-z])M\.?\s*M\.?(?![A-Za-z])", re.IGNORECASE), "母乳"),
    (re.compile(r"(?<![A-Za-z])Compound\s+V(?![A-Za-z])", re.IGNORECASE), "五号化合物"),
    (re.compile(r"(?<![A-Za-z])Gargantua(?![A-Za-z])", re.IGNORECASE), "卡冈图雅"),
    (re.compile(r"(?<![A-Za-z])Lazarus(?![A-Za-z])", re.IGNORECASE), "拉撒路"),
    (re.compile(r"(?<![A-Za-z])TARS(?![A-Za-z])", re.IGNORECASE), "塔斯"),
    (re.compile(r"(?<![A-Za-z])CASE(?![A-Za-z])", re.IGNORECASE), "凯斯"),
]


@dataclass(frozen=True)
class TranslateJobRequest:
    input_path: Path
    output_path: Path | None = None
    provider: str = "gemini"
    base_url: str | None = None
    api_key: str | None = None
    model: str | None = None
    source_language: str | None = None
    target_language: str = "zh"
    style: str | None = None
    replay_path: Path | None = None
    dry_run: bool = False
    json_mode: bool = False


@dataclass(frozen=True)
class CueChunk:
    target_cues: list[SubtitleCue]
    context_before: list[SubtitleCue]
    context_after: list[SubtitleCue]


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Translate and clean a subtitle draft")
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--provider", default="gemini")
    parser.add_argument("--base-url")
    parser.add_argument("--api-key")
    parser.add_argument("--model")
    parser.add_argument("--source-language")
    parser.add_argument("--target-language", default="zh")
    parser.add_argument("--style")
    parser.add_argument("--replay", type=Path, help="Replay a saved Gemini response from a local JSON file")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--json", action="store_true")
    return parser


def _resolve_input_path(path: Path) -> Path:
    if path.is_file():
        if path.suffix.lower() == ".srt":
            return path
        raise ValueError(
            f"subflow translate currently supports only SRT input. Got {path.suffix or '<no suffix>'} at {path}.",
        )
    if path.is_dir():
        candidates = sorted(item for item in path.iterdir() if item.is_file() and item.suffix.lower() == ".srt")
        if candidates:
            return candidates[0]
        raise FileNotFoundError(
            f"No .srt subtitle file found at {path}. subflow translate currently supports only SRT input.",
        )
    raise FileNotFoundError(f"No subtitle or transcript file found at {path}")


def _chunk_cues(cues: list[SubtitleCue], *, max_items: int = 20, max_chars: int = 2800) -> list[CueChunk]:
    chunks: list[CueChunk] = []
    current: list[SubtitleCue] = []
    current_start = 0
    char_count = 0
    for cue_index, cue in enumerate(cues):
        if not current:
            current_start = cue_index
        cue_size = len(cue.text)
        if current and (len(current) >= max_items or char_count + cue_size > max_chars):
            current_end = current_start + len(current)
            chunks.append(
                CueChunk(
                    target_cues=current,
                    context_before=cues[max(0, current_start - CONTEXT_WINDOW_CUES) : current_start],
                    context_after=cues[current_end : min(len(cues), current_end + CONTEXT_WINDOW_CUES)],
                )
            )
            current = []
            char_count = 0
            current_start = cue_index
        current.append(cue)
        char_count += cue_size
    if current:
        current_end = current_start + len(current)
        chunks.append(
            CueChunk(
                target_cues=current,
                context_before=cues[max(0, current_start - CONTEXT_WINDOW_CUES) : current_start],
                context_after=cues[current_end : min(len(cues), current_end + CONTEXT_WINDOW_CUES)],
            )
        )
    return chunks


def _normalize_source_text(text: str) -> str:
    normalized = text.replace("\ufeff", "").replace("\r\n", "\n").replace("\r", "\n")
    normalized = re.sub(r"{\s*([^}]+?)\s*}", r"{\1}", normalized)
    normalized = re.sub(r"\s+([,.:;!?])", r"\1", normalized)
    lines = [line.strip() for line in normalized.split("\n")]
    return "\n".join(lines).strip()


def _strip_markup(text: str) -> str:
    return re.sub(r"<[^>]+>|{[^}]+}", "", text)


def _plain_source_text(cue: SubtitleCue) -> str:
    return _strip_markup(_normalize_source_text(cue.text)).strip()


def _plain_source_line(line: str) -> str:
    return _strip_markup(_normalize_source_text(line)).strip()


def _looks_like_local_path(text: str) -> bool:
    stripped = text.strip()
    return bool(re.match(r"^(?:[A-Za-z]:[\\/]|\\\\)", stripped))


def _looks_like_release_filename(text: str) -> bool:
    if not re.search(rf"\.({'|'.join(IRRELEVANT_FILE_EXTENSIONS)})\b", text, re.IGNORECASE):
        return False
    return _looks_like_local_path(text) or "/" in text or "\\" in text or text.count(".") >= 3


def _is_irrelevant_subtitle_line(line: str) -> bool:
    plain = _plain_source_line(line)
    if not plain:
        return True

    compact = re.sub(r"\s+", "", plain)
    upper = compact.upper()
    if re.fullmatch(r"(?:https?://)?(?:www\.)?[A-Za-z0-9.-]+\.[A-Za-z]{2,}(?:/[^\s]*)?", compact):
        return True
    if "MY-SUBS" in upper:
        return True
    if "EASY SUBTITLES SYNCHRONIZER" in upper:
        return True
    if "REPAIR AND SYNCHRONIZATION BY" in upper:
        return True
    if _looks_like_local_path(plain):
        return True
    if _looks_like_release_filename(plain):
        return True
    return False


def _is_irrelevant_prefix_cue(cue: SubtitleCue) -> bool:
    lines = [line for line in _normalize_source_text(cue.text).split("\n") if line.strip()]
    return bool(lines) and all(_is_irrelevant_subtitle_line(line) for line in lines)


def _sanitize_source_cue_text(text: str) -> str:
    cleaned_lines = [
        line
        for line in _normalize_source_text(text).split("\n")
        if line.strip() and not _is_irrelevant_subtitle_line(line)
    ]
    return "\n".join(cleaned_lines).strip()


def _prepare_cues(cues: list[SubtitleCue]) -> list[SubtitleCue]:
    prepared: list[SubtitleCue] = []
    for cue in cues:
        cleaned_text = _sanitize_source_cue_text(cue.text)
        if not cleaned_text:
            continue
        prepared.append(
            SubtitleCue(
                index=len(prepared) + 1,
                start=cue.start,
                end=cue.end,
                text=cleaned_text,
            )
        )
    return prepared


def _source_lines(text: str) -> list[str]:
    return [line.strip() for line in _normalize_source_text(text).split("\n") if line.strip()]


def _schema() -> dict[str, Any]:
    return {
        "type": "object",
        "properties": {
            "translations": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "id": {"type": "integer"},
                        "lines": {
                            "type": "array",
                            "items": {"type": "string"},
                            "minItems": 1,
                            "maxItems": MAX_SUBTITLE_LINES,
                        },
                    },
                    "required": ["id", "lines"],
                },
            },
        },
        "required": ["translations"],
    }


def _style_requests_no_punctuation(style: str | None) -> bool:
    if not style:
        return False
    normalized = style.lower()
    tokens = [
        "no punctuation",
        "without punctuation",
        "no-punctuation",
        "strip punctuation",
        "无标点",
        "不要标点",
        "不加标点",
        "去标点",
    ]
    return any(token in normalized for token in tokens)


_STATE_CUE_PATTERN = re.compile(r"([（(\[])\s*([^()\[\]（）]+?)\s*([）)\]])")


def _replace_punctuation_with_spaces_preserving_states(text: str) -> str:
    parts = re.split(r"(<[^>]+>|{[^}]+})", text)
    cleaned: list[str] = []
    for part in parts:
        if not part:
            continue
        if re.fullmatch(r"<[^>]+>|{[^}]+}", part):
            cleaned.append(part)
            continue

        state_placeholders: list[str] = []

        def _protect_state(match: re.Match[str]) -> str:
            content = re.sub(r"\s+", " ", match.group(2)).strip()
            if not content:
                return " "
            state_placeholders.append(f"（{content}）")
            return f"SUBFLOWSTATETOKEN{len(state_placeholders) - 1}X"

        protected = _STATE_CUE_PATTERN.sub(_protect_state, part)
        replaced = "".join(
            " " if unicodedata.category(char).startswith("P") else char
            for char in protected
        )
        replaced = re.sub(r"\s+", " ", replaced).strip()
        for index, state_text in enumerate(state_placeholders):
            replaced = replaced.replace(f"SUBFLOWSTATETOKEN{index}X", state_text)
        cleaned.append(replaced)
    return "".join(cleaned)


def _wrap_plain_text(text: str, *, target_width: int, max_lines: int) -> list[str]:
    if not text:
        return []
    words = text.split()
    if len(words) > 1:
        lines: list[str] = []
        current = ""
        for word in words:
            candidate = f"{current} {word}".strip()
            if current and display_width(candidate) > target_width and len(lines) + 1 < max_lines:
                lines.append(current)
                current = word
            else:
                current = candidate
        if current:
            lines.append(current)
        return lines

    lines = []
    current = ""
    for char in text:
        candidate = current + char
        if current and display_width(candidate) > target_width and len(lines) + 1 < max_lines:
            lines.append(current)
            current = char
        else:
            current = candidate
    if current:
        lines.append(current)
    return lines


def _rebalance_lines(lines: list[str], cue: SubtitleCue) -> list[str]:
    normalized = [line.strip() for line in lines if line.strip()]
    if not normalized:
        return []
    if len(normalized) > MAX_SUBTITLE_LINES:
        normalized = [" ".join(normalized)]

    source_line_count = max(1, min(len(_source_lines(cue.text)), MAX_SUBTITLE_LINES))
    max_lines = min(MAX_SUBTITLE_LINES, source_line_count if source_line_count > 1 else MAX_SUBTITLE_LINES)

    if len(normalized) == 1:
        single = normalized[0]
        if display_width(_strip_markup(single)) > TARGET_LINE_WIDTH:
            wrapped = _wrap_plain_text(single, target_width=TARGET_LINE_WIDTH, max_lines=max_lines)
            if wrapped:
                normalized = wrapped

    return normalized[:MAX_SUBTITLE_LINES]


def _normalize_known_named_entities(text: str) -> str:
    normalized = text
    for pattern, replacement in KNOWN_NAMED_ENTITY_REPLACEMENTS:
        normalized = pattern.sub(replacement, normalized)
    return normalized


def _write_chunk_debug_artifacts(
    chunk_dir: Path,
    chunk_index: int,
    prompt: str,
    payload: dict[str, Any],
    *,
    variant: str = "",
) -> None:
    chunk_dir.mkdir(parents=True, exist_ok=True)
    suffix = f".{variant}" if variant else ""
    prefix = f"chunk-{chunk_index:03d}{suffix}"
    (chunk_dir / f"{prefix}.prompt.txt").write_text(prompt, encoding="utf-8")

    raw_payload = payload.get("__raw_api_payload")
    if raw_payload is not None:
        (chunk_dir / f"{prefix}.response.raw.json").write_text(
            json.dumps(raw_payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    raw_completion_text = payload.get("__raw_completion_text")
    if isinstance(raw_completion_text, str) and raw_completion_text.strip():
        (chunk_dir / f"{prefix}.response.content.txt").write_text(raw_completion_text, encoding="utf-8")

    normalized_payload = {key: value for key, value in payload.items() if key.startswith("__") is False}
    (chunk_dir / f"{prefix}.response.normalized.json").write_text(
        json.dumps(normalized_payload, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def _contains_chinese(text: str) -> bool:
    return bool(re.search(r"[\u4e00-\u9fff]", text))


def _contains_ascii_letters(text: str) -> bool:
    return bool(re.search(r"[A-Za-z]", text))


def _has_untranslated_label_fragment(text: str) -> bool:
    stripped = text.strip()
    if not stripped:
        return False

    patterns = [
        r'^(?:[-—–]\s*)?[\[\(（【<]\s*[A-Z][A-Za-z]+(?:\s+[A-Z][A-Za-z]+){0,2}(?:\s+[A-Z][A-Za-z]+)?\s*[\]\)）】>]\s*',
        r'^(?:[-—–]\s*)?[A-Z][A-Za-z]+(?:\s+[A-Z][A-Za-z]+){0,2}\s*[:：]\s*',
    ]
    return any(re.match(pattern, stripped) for pattern in patterns)


def _is_allowed_english_only_line(text: str, cue: SubtitleCue) -> bool:
    plain = _plain_source_line(text)
    if not plain:
        return True
    if _is_irrelevant_subtitle_line(plain):
        return True

    words = re.findall(r"[A-Za-z]+(?:'[A-Za-z]+)?", plain)
    compact = re.sub(r"[^A-Za-z]", "", plain)
    if not words:
        return True

    if len(words) == 1:
        word = words[0]
        if (
            len(word) >= 2
            and len(word) <= 6
            and word.upper() == word
            and re.fullmatch(r"[A-Z]{2,6}", plain) is not None
        ):
            return True

    cue_kind = _cue_kind(cue)
    if cue_kind != "dialogue" and len(compact) >= 2 and len(compact) <= 8 and len(words) <= 2 and plain.upper() == plain:
        return True

    return False


def _apply_default_style(request: TranslateJobRequest) -> TranslateJobRequest:
    if request.style:
        return request
    settings = load_subflow_config()
    if not settings.translate_style:
        return request
    return replace(request, style=settings.translate_style)


def _normalize_provider(provider: str | None) -> str:
    return (provider or "gemini").strip().lower() or "gemini"


def _resolve_translate_api_key(request: TranslateJobRequest) -> str | None:
    if request.api_key and request.api_key.strip():
        return request.api_key.strip()
    return load_subflow_config().api_key


def _resolve_translate_base_url(request: TranslateJobRequest) -> str | None:
    if request.base_url and request.base_url.strip():
        return request.base_url.strip()
    return load_subflow_config().base_url


def _use_openai_compatible_transport(provider: str, request: TranslateJobRequest) -> bool:
    return bool(_resolve_translate_base_url(request)) or provider != "gemini"


def _dry_run_payload(request: TranslateJobRequest) -> dict[str, Any]:
    provider = _normalize_provider(request.provider)
    use_openai_compatible = _use_openai_compatible_transport(provider, request)
    return {
        "status": "planned",
        "provider": provider,
        "transport": "openai-compatible" if use_openai_compatible else "gemini-native",
        "base_url": _resolve_translate_base_url(request),
        "model": request.model,
        "input": str(request.input_path),
        "output": str(request.output_path) if request.output_path else None,
        "source_language": request.source_language,
        "target_language": request.target_language,
        "style": request.style,
        "requires": ["api_key", "base_url"] if use_openai_compatible else ["api_key"],
    }


def _load_replay_payload(path: Path) -> dict[str, Any]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError("Translate replay file must contain a JSON object.")
    return payload


def _request_translation_payload(
    client: Any | None,
    *,
    provider: str,
    model: str,
    prompt: str,
    request: TranslateJobRequest,
) -> dict[str, Any]:
    if _use_openai_compatible_transport(provider, request):
        api_key = _resolve_translate_api_key(request)
        base_url = _resolve_translate_base_url(request)
        if not api_key:
            raise OpenAICompatibleError(
                "Set SUBFLOW_TRANSLATE_API_KEY or OPENAI_API_KEY, or configure translate_api_key/api_key in subflow config before running translate.",
            )
        if not base_url:
            raise OpenAICompatibleError(
                "Set SUBFLOW_TRANSLATE_BASE_URL or OPENAI_BASE_URL, or configure translate_base_url/base_url in subflow config before running translate.",
            )
        return generate_openai_json_response(
            base_url=base_url,
            api_key=api_key,
            model=model,
            prompt=prompt,
            schema=_schema(),
        )
    if client is None:
        raise GeminiUnavailableError(
            "Set SUBFLOW_TRANSLATE_API_KEY or GEMINI_API_KEY, or configure api_key in subflow config before running subflow translate.",
        )
    return generate_json_response(
        client,
        model=model,
        contents=[prompt],
        schema=_schema(),
    )


def _translation_lines_from_item(item: dict[str, Any]) -> list[str]:
    lines = item.get("lines")
    if isinstance(lines, list):
        return [str(line).strip() for line in lines if str(line).strip()]

    text = item.get("text")
    if isinstance(text, str) and text.strip():
        return [segment.strip() for segment in text.replace("\r\n", "\n").replace("\r", "\n").split("\n") if segment.strip()]

    raise RuntimeError(f"Gemini translation item for cue {item.get('id')} did not contain lines or text.")


def _translation_text_from_item(item: dict[str, Any], request: TranslateJobRequest, cue: SubtitleCue) -> str:
    lines = _rebalance_lines(_translation_lines_from_item(item), cue)
    return _postprocess_translation_text("\n".join(lines), request)


def _cue_kind(cue: SubtitleCue) -> str:
    plain = _plain_source_text(cue)
    if "<i>" in cue.text.lower():
        return "italic_dialogue_or_lyric"
    if plain and "\n" not in plain:
        letters = [char for char in plain if char.isalpha()]
        looks_like_dialogue_punctuation = bool(re.search(r"[?!.,;:!?]$", plain))
        if letters and plain.upper() == plain and len(plain.split()) <= 5 and not looks_like_dialogue_punctuation:
            return "screen_text"
        if letters and len(letters) >= 4:
            upper_ratio = sum(1 for char in letters if char.isupper()) / len(letters)
            if upper_ratio >= 0.8 and not looks_like_dialogue_punctuation:
                return "system_or_broadcast"
    if plain and re.fullmatch(r"[\[(].+[\])]", plain):
        return "screen_text"
    return "dialogue"


def _cleanup_translation_line(text: str) -> str:
    cleaned = text.replace("\t", " ").strip()
    cleaned = re.sub(r"\s*[/\\|]+\s*([,.;:!?。，！？；：])", r"\1", cleaned)
    cleaned = re.sub(r"([,.;:!?。，！？；：])\s*[/\\|]+\s*$", r"\1", cleaned)
    cleaned = re.sub(r"^\s*[/\\|]+\s*", "", cleaned)
    cleaned = re.sub(r"\s*[/\\|]+\s*$", "", cleaned)
    cleaned = re.sub(r"\s+([,.;:!?。，！？；：])", r"\1", cleaned)
    cleaned = re.sub(r"([(\[<{])\s+", r"\1", cleaned)
    cleaned = re.sub(r"\s+([)\]>}])", r"\1", cleaned)
    cleaned = re.sub(r"\s{2,}", " ", cleaned)
    return cleaned.strip()


def _is_punctuation_only_line(text: str) -> bool:
    visible = _strip_markup(text).strip()
    if not visible:
        return True
    return re.fullmatch(r"[\W_]+", visible, re.UNICODE) is not None


def _postprocess_translation_text(text: str, request: TranslateJobRequest) -> str:
    normalized = text.replace("\r\n", "\n").replace("\r", "\n").strip()
    normalized = _normalize_source_text(normalized)
    lines = [line.strip() for line in normalized.split("\n") if line.strip()]
    lines = [line for line in lines if not _is_irrelevant_subtitle_line(line)]
    lines = [_cleanup_translation_line(line) for line in lines]
    if _style_requests_no_punctuation(request.style):
        lines = [_replace_punctuation_with_spaces_preserving_states(line).strip() for line in lines]
    lines = [line for line in lines if line and not _is_punctuation_only_line(line)]
    lines = [_normalize_known_named_entities(line) for line in lines]
    return "\n".join(line for line in lines if line).strip()


def _cue_prompt_payload(cue: SubtitleCue, *, role: str = "target") -> dict[str, Any]:
    return {
        "id": cue.index,
        "role": role,
        "kind": _cue_kind(cue),
        "preferred_line_count": max(1, min(len(_source_lines(cue.text)), MAX_SUBTITLE_LINES)),
        "source_lines": _source_lines(cue.text),
    }


def _has_suspicious_mixed_script_dialogue(text: str, cue: SubtitleCue) -> bool:
    if _cue_kind(cue) != "dialogue":
        return False
    if _contains_chinese(text) is False or _contains_ascii_letters(text) is False:
        return False
    visible = _strip_markup(text)
    for token in re.findall(r"[A-Za-z][A-Za-z0-9'.-]*", visible):
        letters = re.sub(r"[^A-Za-z]", "", token)
        if not letters:
            continue
        if len(letters) == 1:
            return True
        if letters.upper() == letters and 2 <= len(letters) <= 6:
            continue
        return True
    return False


def _needs_untranslated_repair(text: str, cue: SubtitleCue) -> bool:
    lines = [line.strip() for line in text.split("\n") if line.strip()]
    for line in lines:
        if not _contains_ascii_letters(line):
            continue
        if _has_untranslated_label_fragment(line):
            return True
        if _has_suspicious_mixed_script_dialogue(line, cue):
            return True
        if _contains_chinese(line):
            continue
        if _is_allowed_english_only_line(line, cue):
            continue
        return True
    return False


def _repair_prompt_payload(
    cue: SubtitleCue,
    current_text: str,
    cue_lookup: dict[int, SubtitleCue] | None = None,
) -> dict[str, Any]:
    cue_lookup = cue_lookup or {cue.index: cue}
    previous_cue = cue_lookup.get(cue.index - 1)
    next_cue = cue_lookup.get(cue.index + 1)
    return {
        "id": cue.index,
        "kind": _cue_kind(cue),
        "source_lines": _source_lines(cue.text),
        "previous_source_lines": _source_lines(previous_cue.text) if previous_cue is not None else [],
        "next_source_lines": _source_lines(next_cue.text) if next_cue is not None else [],
        "current_lines": [line.strip() for line in current_text.split("\n") if line.strip()],
    }


def _build_repair_prompt(
    request: TranslateJobRequest,
    cue_texts: list[tuple[SubtitleCue, str]],
    cue_lookup: dict[int, SubtitleCue] | None = None,
) -> str:
    lines = [
        f"Repair the following subtitle cues into clean {request.target_language} subtitle text.",
        "The previous translation left some English unchanged or structurally awkward.",
        "Rules:",
        "1. Return exactly one item for every cue id below. Do not drop, merge, reorder, or skip any cue.",
        "2. Rewrite every remaining English dialogue fragment into natural Chinese. Do not keep English sentences unchanged.",
        "3. Proper names and acronyms may stay in their original script only when Chinese viewers normally expect that exact form. For recurring spoken names or named objects such as Tom, Murph, Hughie, Translucent, Gargantua, Lazarus, TARS, or CASE, prefer the standard Chinese transliteration or conventional Chinese rendering when local context makes it clear.",
        "4. If a cue is only a speaker label or on-screen label, convert it into concise Chinese bracketed form and do not leave a raw English-only or mixed English-label form unchanged.",
        "5. Repair clipped OCR fragments, mixed Chinese-English name leakage, and awkward punctuation debris into concise natural Chinese.",
        "6. Keep globally familiar institutional or technical acronyms such as NASA, GPS, FBI, CIA, USB, AI, and RPM only when Chinese subtitle readers commonly recognize them in that raw form and the raw acronym is not the whole spoken subtitle line by itself.",
        "7. Use previous_source_lines and next_source_lines only to recover damaged fragments or stabilize names. Do not borrow full meaning from neighboring cues, do not repeat neighboring lines early, and do not rewrite an already good cue just for style.",
        "8. Remove watermark, synchronization-credit, and release-noise text completely while still returning a valid subtitle line when the cue carries scene meaning.",
        "9. Keep the subtitle concise and return a JSON array only. Each item must be {\"id\": <int>, \"lines\": [<line1>, <optional line2>]} with no commentary.",
        "CUES:",
    ]
    local_lookup = cue_lookup or {cue.index: cue for cue, _current_text in cue_texts}
    for cue, current_text in cue_texts:
        lines.append(json.dumps(_repair_prompt_payload(cue, current_text, local_lookup), ensure_ascii=False))
    return "\n".join(lines)


def _repair_chunk_translations(
    client: Any | None,
    *,
    provider: str,
    model: str,
    request: TranslateJobRequest,
    cue_texts: list[tuple[SubtitleCue, str]],
    cue_lookup: dict[int, SubtitleCue] | None = None,
    chunk_index: int,
    chunk_debug_dir: Path | None,
) -> dict[int, str]:
    prompt = _build_repair_prompt(request, cue_texts, cue_lookup)
    cue_by_id = {cue.index: cue for cue, _text in cue_texts}
    repaired: dict[int, str] = {cue.index: text for cue, text in cue_texts}

    for attempt in range(1, 3):
        payload = _request_translation_payload(
            client,
            provider=provider,
            model=model,
            prompt=prompt,
            request=request,
        )
        if chunk_debug_dir is not None:
            _write_chunk_debug_artifacts(
                chunk_debug_dir,
                chunk_index,
                prompt,
                payload,
                variant=f"repair-{attempt}",
            )

        items = payload.get("translations", [])
        if not isinstance(items, list):
            raise RuntimeError("Repair translation response did not contain translations.")

        repaired = {}
        for item in items:
            cue_id = int(item["id"])
            cue = cue_by_id.get(cue_id)
            if cue is None:
                continue
            repaired[cue_id] = _translation_text_from_item(item, request, cue)

        remaining = [
            (cue_by_id[cue_id], text)
            for cue_id, text in repaired.items()
            if cue_id in cue_by_id and text and _needs_untranslated_repair(text, cue_by_id[cue_id])
        ]
        if not remaining:
            break
        prompt = _build_repair_prompt(request, remaining, cue_lookup)
    return repaired


def _coerce_chunk(cues: CueChunk | list[SubtitleCue]) -> CueChunk:
    if isinstance(cues, CueChunk):
        return cues
    return CueChunk(target_cues=list(cues), context_before=[], context_after=[])


def _build_prompt(request: TranslateJobRequest, cues: CueChunk | list[SubtitleCue]) -> str:
    chunk = _coerce_chunk(cues)
    lines = [
        f"Translate the following subtitle cues from {request.source_language or 'the source language'} to {request.target_language}.",
        "This is subtitle translation, not prose translation. The source may be noisy OCR English or awkward intermediate English.",
        "Rules:",
        "1. Return exactly one item for every cue id below. Only cue ids marked as target cues should be returned. Do not drop, merge, reorder, or skip any target cue. Do not return items for context_only cues.",
        "2. Translate every meaningful target cue into natural spoken Chinese. Do not leave an English sentence unchanged. If unsure, give the safest Chinese paraphrase instead of copying the English source.",
        "3. Exceptions: keep proper names, acronyms, HTML markup, and inline control markers such as {n8} unchanged only when Chinese viewers normally expect that exact raw form. However, a bare speaker label, a line-leading English label, or an isolated acronym or alphanumeric shorthand used as the whole spoken cue must not remain as raw English text. For recurring spoken character names, family given names, nicknames, places, spacecraft, missions, or named objects such as Tom, Murph, Hughie, Translucent, Gargantua, Lazarus, TARS, or CASE, prefer the standard Chinese transliteration or conventional Chinese rendering when local context makes it obvious.",
        "4. Very short cues still need translation when they carry meaning: questions, calls, shouts, fragments, trailing phrases, confirmations, and commands.",
        "5. If one sentence is split across neighboring target cues, translate each cue as the matching fragment of that same sentence. Keep unfinished openings naturally unfinished when the source is unfinished.",
        "6. Some entries below are marked context_only. Use them only to disambiguate damaged text or unfinished clauses for nearby target cues. Do not translate context_only entries, do not pull words, facts, conclusions, or emotional color from another cue into the current cue, and do not anticipate the next cue or repeat it early.",
        "7. When the source is damaged, repair the smallest span necessary from nearby cues. Choose the most conservative scene-fitting reading and avoid creative rewrites.",
        "8. Preserve meaning, speaker intent, tone, polarity, and plot facts. Do not invert meaning or add motives, sarcasm, or stronger wording that is not present.",
        "9. Keep concrete nouns, objects, destinations, and requests concrete. Do not smooth them into vague or abstract Chinese.",
        "10. Prefer concise natural subtitle Chinese over mirrored English word order. Compress only when needed for subtitle readability.",
        "11. Translate on-screen labels, lyrics, and stylized lines into concise readable Chinese. Keep non-dialogue cues visibly distinct with bracketed Chinese such as [sound] style cues translated into Chinese brackets.",
        "12. Never leave broken OCR letter fragments or clipped name tails unchanged. Repair pieces such as S., T., A., Y., or rph into natural Chinese from the local cue context.",
        "13. Keep globally familiar institutional or technical acronyms such as NASA, GPS, FBI, CIA, USB, AI, and RPM only when Chinese subtitle readers commonly recognize them in that raw form and the cue is not just that raw acronym or shorthand by itself. Short spoken cues such as NASA, RPM, 1G, or 2G should be rendered into natural Chinese from context unless they are clearly literal on-screen codes. Do not use this exception to leave ordinary dialogue names untranslated.",
        "14. Drop local file paths, release filenames, site watermarks, URL-only lines, and release-group prefixes unrelated to the scene.",
        "15. Each item's lines array must contain 1 or 2 subtitle lines only, short enough to read on screen.",
        "16. Return a JSON array only. Each item must be {\"id\": <int>, \"lines\": [<line1>, <optional line2>]} with no commentary.",
    ]
    if request.style:
        lines.append(f"Follow this style guide exactly after preserving the meaning: {request.style}")
    if _style_requests_no_punctuation(request.style):
        lines.append(
            "When the style requests no punctuation, replace ordinary dialogue punctuation with spaces instead of deleting words, but keep bracketed state cues intact.",
        )
    lines.append("CUES:")
    for cue in chunk.context_before:
        lines.append(json.dumps(_cue_prompt_payload(cue, role='context_only'), ensure_ascii=False))
    for cue in chunk.target_cues:
        lines.append(json.dumps(_cue_prompt_payload(cue, role='target'), ensure_ascii=False))
    for cue in chunk.context_after:
        lines.append(json.dumps(_cue_prompt_payload(cue, role='context_only'), ensure_ascii=False))
    return "\n".join(lines)


def _translate_chunks(
    client: Any | None,
    *,
    model: str,
    request: TranslateJobRequest,
    chunks: list[CueChunk],
) -> dict[int, str]:
    translated: dict[int, str] = {}
    chunk_debug_dir = (request.output_path.parent / "chunk-debug") if request.output_path else None
    provider = _normalize_provider(request.provider)
    for chunk_index, chunk in enumerate(chunks, start=1):
        prompt = _build_prompt(request, chunk)
        cue_by_id = {cue.index: cue for cue in chunk.target_cues}
        repair_lookup = {cue.index: cue for cue in [*chunk.context_before, *chunk.target_cues, *chunk.context_after]}
        last_error: Exception | None = None
        payload: dict[str, Any] | None = None
        for _attempt in range(3):
            try:
                payload = _request_translation_payload(
                    client,
                    provider=provider,
                    model=model,
                    prompt=prompt,
                    request=request,
                )
                break
            except Exception as exc:
                last_error = exc
        if payload is None:
            assert last_error is not None
            raise last_error
        if chunk_debug_dir is not None:
            _write_chunk_debug_artifacts(chunk_debug_dir, chunk_index, prompt, payload)
        items = payload.get("translations", [])
        if not isinstance(items, list):
            raise RuntimeError("Gemini translation response did not contain translations.")
        for item in items:
            cue_id = int(item["id"])
            cue = cue_by_id.get(cue_id)
            if cue is None:
                raise RuntimeError(f"Gemini returned unexpected cue id {cue_id}.")
            translated[cue_id] = _translation_text_from_item(item, request, cue)

        repair_candidates: list[tuple[SubtitleCue, str]] = []
        for cue in chunk.target_cues:
            current_text = translated.get(cue.index, "")
            if current_text and _needs_untranslated_repair(current_text, cue):
                repair_candidates.append((cue, current_text))
        if repair_candidates:
            try:
                repaired = _repair_chunk_translations(
                    client,
                    provider=provider,
                    model=model,
                    request=request,
                    cue_texts=repair_candidates,
                    cue_lookup=repair_lookup,
                    chunk_index=chunk_index,
                    chunk_debug_dir=chunk_debug_dir,
                )
                for cue_id, repaired_text in repaired.items():
                    translated[cue_id] = repaired_text
            except Exception:
                pass
    return translated


def run_translate_job(request: TranslateJobRequest) -> dict[str, Any]:
    request = _apply_default_style(request)
    request = replace(request, provider=_normalize_provider(request.provider))
    if request.dry_run:
        return _dry_run_payload(request)

    source = _resolve_input_path(request.input_path)
    output_path = request.output_path or source.with_suffix(".zh.srt")
    cues = _prepare_cues(read_srt(source))
    cue_by_id = {cue.index: cue for cue in cues}
    if request.replay_path is not None:
        payload = _load_replay_payload(request.replay_path)
        items = payload.get("translations", [])
        if not isinstance(items, list):
            raise RuntimeError("Translate replay file must contain a translations array.")
        translations = {}
        for item in items:
            cue_id = int(item["id"])
            cue = cue_by_id.get(cue_id)
            if cue is None:
                continue
            text = _translation_text_from_item(item, request, cue)
            translations[cue_id] = text
        output_cues: list[SubtitleCue] = []
        for cue in cues:
            if cue.index in translations:
                translated_text = translations[cue.index]
                if not translated_text:
                    continue
            else:
                translated_text = cue.text
            output_cues.append(SubtitleCue(index=cue.index, start=cue.start, end=cue.end, text=translated_text))
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(write_srt(output_cues), encoding="utf-8")
        sidecar = output_path.with_suffix(".json")
        sidecar.write_text(
            json.dumps(
                {
                    "provider": request.provider,
                    "model": request.model,
                    "source_language": request.source_language,
                    "target_language": request.target_language,
                    "style": request.style,
                    "source": str(source),
                    "replay": str(request.replay_path),
                    "translations": translations,
                },
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        return {
            "status": "replayed",
            "provider": request.provider,
            "model": request.model,
            "input": str(source),
            "output": str(output_path),
            "sidecar": str(sidecar),
            "cue_count": len(output_cues),
            "replay": str(request.replay_path),
        }

    settings = load_subflow_config()
    model = request.model or settings.translate_model
    client = None
    if _use_openai_compatible_transport(request.provider, request) is False:
        if not _resolve_translate_api_key(request):
            raise GeminiUnavailableError(
                "Set SUBFLOW_TRANSLATE_API_KEY or GEMINI_API_KEY, or configure api_key in subflow config before running subflow translate.",
            )
        client = create_client(settings)
    translations = _translate_chunks(client, model=model, request=request, chunks=_chunk_cues(cues))

    output_cues: list[SubtitleCue] = []
    for cue in cues:
        if cue.index in translations:
            translated_text = translations[cue.index]
            if not translated_text:
                continue
        else:
            translated_text = cue.text
        output_cues.append(SubtitleCue(index=cue.index, start=cue.start, end=cue.end, text=translated_text))

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(write_srt(output_cues), encoding="utf-8")
    sidecar = output_path.with_suffix(".json")
    sidecar.write_text(
        json.dumps(
            {
                "provider": request.provider,
                "model": model,
                "source_language": request.source_language,
                "target_language": request.target_language,
                "style": request.style,
                "source": str(source),
                "translations": translations,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    return {
        "status": "ok",
        "provider": request.provider,
        "model": model,
        "input": str(source),
        "output": str(output_path),
        "sidecar": str(sidecar),
        "cue_count": len(output_cues),
    }


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    request = TranslateJobRequest(
        input_path=args.input,
        output_path=args.output,
        provider=args.provider,
        base_url=args.base_url,
        api_key=args.api_key,
        model=args.model,
        source_language=args.source_language,
        target_language=args.target_language,
        style=args.style,
        replay_path=args.replay,
        dry_run=args.dry_run,
        json_mode=args.json,
    )
    try:
        result = run_translate_job(request)
    except (GeminiUnavailableError, OpenAICompatibleError) as exc:
        if args.json:
            print(json.dumps({"status": "error", "error": str(exc)}, ensure_ascii=False, indent=2))
        else:
            print(str(exc), file=sys.stderr)
        return 2
    except Exception as exc:
        if args.json:
            print(json.dumps({"status": "error", "error": str(exc)}, ensure_ascii=False, indent=2))
        else:
            print(str(exc), file=sys.stderr)
        return 1

    if args.json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        if result.get("status") == "planned":
            print(f"Planned translation job for {args.input}")
        else:
            print(f"Wrote {result['cue_count']} translated cues to {result['output']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
