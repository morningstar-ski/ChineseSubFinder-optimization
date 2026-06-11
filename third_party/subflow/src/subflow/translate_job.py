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
from .subtitle_io import SubtitleCue, display_width, read_srt, write_srt


SUPPORTED_TRANSLATE_PROVIDERS = {"gemini"}
MAX_SUBTITLE_LINES = 2
TARGET_LINE_WIDTH = 22
IRRELEVANT_FILE_EXTENSIONS = ("srt", "ass", "ssa", "sub", "idx", "sup", "vtt")


@dataclass(frozen=True)
class TranslateJobRequest:
    input_path: Path
    output_path: Path | None = None
    provider: str = "gemini"
    model: str | None = None
    source_language: str | None = None
    target_language: str = "zh"
    style: str | None = None
    replay_path: Path | None = None
    dry_run: bool = False
    json_mode: bool = False


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Translate and clean a subtitle draft")
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--provider", default="gemini")
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


def _chunk_cues(cues: list[SubtitleCue], *, max_items: int = 20, max_chars: int = 2800) -> list[list[SubtitleCue]]:
    chunks: list[list[SubtitleCue]] = []
    current: list[SubtitleCue] = []
    char_count = 0
    for cue in cues:
        cue_size = len(cue.text)
        if current and (len(current) >= max_items or char_count + cue_size > max_chars):
            chunks.append(current)
            current = []
            char_count = 0
        current.append(cue)
        char_count += cue_size
    if current:
        chunks.append(current)
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


def _cue_kind(cue: SubtitleCue) -> str:
    plain = _plain_source_text(cue)
    if plain and "\n" not in plain:
        letters = [char for char in plain if char.isalpha()]
        if letters and plain.upper() == plain and len(plain.split()) <= 5:
            return "screen_text"
    if "<i>" in cue.text.lower():
        return "italic_dialogue_or_lyric"
    return "dialogue"


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


def _postprocess_translation_text(text: str, request: TranslateJobRequest) -> str:
    normalized = text.replace("\r\n", "\n").replace("\r", "\n").strip()
    normalized = _normalize_source_text(normalized)
    lines = [line.strip() for line in normalized.split("\n") if line.strip()]
    lines = [line for line in lines if not _is_irrelevant_subtitle_line(line)]
    if _style_requests_no_punctuation(request.style):
        lines = [_replace_punctuation_with_spaces_preserving_states(line).strip() for line in lines]
    return "\n".join(line for line in lines if line).strip()


def _cue_prompt_payload(cue: SubtitleCue) -> dict[str, Any]:
    return {
        "id": cue.index,
        "kind": _cue_kind(cue),
        "preferred_line_count": max(1, min(len(_source_lines(cue.text)), MAX_SUBTITLE_LINES)),
        "source_lines": _source_lines(cue.text),
    }


def _build_prompt(request: TranslateJobRequest, cues: list[SubtitleCue]) -> str:
    lines = [
        f"Translate the following subtitle cues from {request.source_language or 'the source language'} to {request.target_language}.",
        "The source file is a noisy intermediate-English subtitle. Some lines may be OCR-damaged, machine-translated from another language, or split awkwardly.",
        "Default to a faithful meaning-first translation. Stay close to the original proposition, action, and relationship.",
        "Use freer adaptation only when a literal rendering would confuse a Chinese viewer because of culture, idiom, joke timing, or obviously broken source text.",
        "When the source is damaged, repair the smallest span necessary from nearby cues. Prefer the most ordinary scene-fitting conversational meaning, not a word-by-word rescue of broken phrasing and not a creative rewrite.",
        "If multiple readings are possible, choose the most conservative one supported by the cue and nearby context.",
        "Do not omit concrete nouns, objects, destinations, or asks just to make the line smoother. If the exact object is uncertain but clearly concrete, keep the Chinese concrete and narrow rather than vague or abstract.",
        "A broken shared-item invitation should stay a shared-item invitation in Chinese, not become a vague line such as 分一下.",
        "Preserve meaning, speaker intent, tone, polarity, joke strength, and plot facts.",
        "Do not invert meaning. Do not turn a question into a statement or a statement into a question.",
        "Do not add hidden motives, sexual implications, sarcasm, abstract wording, or stronger wording unless the source clearly says so.",
        "Prefer natural spoken Chinese over mirrored English word order. When the source is plain, keep the Chinese plain.",
        "Translate on-screen labels and signs into concise Chinese that a viewer can read instantly.",
        "Translate lyrics or stylized lines into concise natural Chinese instead of leaving them as broken literal English fragments.",
        "Keep non-dialogue cues such as applause, laughter, silence, music, ambience, and performance notes visibly distinct from dialogue by using bracketed Chinese such as （掌声）.",
        "If a state cue is attached to dialogue, keep the dialogue readable first and append the state cue in brackets instead of dropping it.",
        "Keep names, HTML markup, and inline control markers such as {n8} unchanged.",
        "Drop local file paths, release filenames, site watermarks, URL-only lines, and release-group prefixes unrelated to the scene instead of translating them.",
        "Return JSON only. For each cue, return one object with id and lines, where lines is an array of 1 or 2 subtitle lines.",
        "Each subtitle line should be concise and readable on screen. Do not output explanations or notes.",
    ]
    if request.style:
        lines.append(f"Follow this style guide exactly after preserving the meaning: {request.style}")
    if _style_requests_no_punctuation(request.style):
        lines.append(
            "When the style requests no punctuation, replace ordinary dialogue punctuation with spaces instead of deleting words, but keep bracketed state cues such as （掌声） intact.",
        )
    lines.append("CUES:")
    for cue in cues:
        lines.append(json.dumps(_cue_prompt_payload(cue), ensure_ascii=False))
    return "\n".join(lines)


def _apply_default_style(request: TranslateJobRequest) -> TranslateJobRequest:
    if request.style:
        return request
    settings = load_subflow_config()
    if not settings.translate_style:
        return request
    return replace(request, style=settings.translate_style)


def _dry_run_payload(request: TranslateJobRequest) -> dict[str, Any]:
    return {
        "status": "planned",
        "provider": request.provider,
        "model": request.model,
        "input": str(request.input_path),
        "output": str(request.output_path) if request.output_path else None,
        "source_language": request.source_language,
        "target_language": request.target_language,
        "style": request.style,
        "requires": ["GEMINI_API_KEY"],
    }


def _load_replay_payload(path: Path) -> dict[str, Any]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError("Translate replay file must contain a JSON object.")
    return payload


def _request_translation_payload(
    client: Any | None,
    *,
    model: str,
    prompt: str,
) -> dict[str, Any]:
    if client is None:
        raise GeminiUnavailableError(
            "Set GEMINI_API_KEY or configure api_key in subflow config before running subflow translate.",
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


def _translate_chunks(
    client: Any | None,
    *,
    model: str,
    request: TranslateJobRequest,
    chunks: list[list[SubtitleCue]],
) -> dict[int, str]:
    translated: dict[int, str] = {}
    for chunk in chunks:
        prompt = _build_prompt(request, chunk)
        cue_by_id = {cue.index: cue for cue in chunk}
        last_error: Exception | None = None
        payload: dict[str, Any] | None = None
        for _attempt in range(3):
            try:
                payload = _request_translation_payload(client, model=model, prompt=prompt)
                break
            except Exception as exc:
                last_error = exc
        if payload is None:
            assert last_error is not None
            raise last_error
        items = payload.get("translations", [])
        if not isinstance(items, list):
            raise RuntimeError("Gemini translation response did not contain translations.")
        for item in items:
            cue_id = int(item["id"])
            cue = cue_by_id.get(cue_id)
            if cue is None:
                raise RuntimeError(f"Gemini returned unexpected cue id {cue_id}.")
            text = _translation_text_from_item(item, request, cue)
            if text:
                translated[cue_id] = text
    return translated


def run_translate_job(request: TranslateJobRequest) -> dict[str, Any]:
    request = _apply_default_style(request)
    if request.dry_run:
        return _dry_run_payload(request)
    if request.provider.lower() not in SUPPORTED_TRANSLATE_PROVIDERS:
        raise ValueError(f"Unsupported translation provider {request.provider!r}. Use gemini.")

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
            if text:
                translations[cue_id] = text
        output_cues: list[SubtitleCue] = []
        for cue in cues:
            translated_text = translations.get(cue.index, cue.text)
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
    if not settings.api_key:
        raise GeminiUnavailableError(
            "Set GEMINI_API_KEY or configure api_key in subflow config before running subflow translate.",
        )
    client = create_client(settings)
    translations = _translate_chunks(client, model=model, request=request, chunks=_chunk_cues(cues))

    output_cues: list[SubtitleCue] = []
    for cue in cues:
        translated_text = translations.get(cue.index, cue.text)
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
    except GeminiUnavailableError as exc:
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
            print(f"Planned Gemini translation job for {args.input}")
        else:
            print(f"Wrote {result['cue_count']} translated cues to {result['output']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
