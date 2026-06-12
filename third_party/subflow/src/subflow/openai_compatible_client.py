from __future__ import annotations

import json
import re
import time
from typing import Any
from urllib import error, request
from urllib.parse import urlparse


class OpenAICompatibleError(RuntimeError):
    pass


def _normalize_endpoint(base_url: str) -> str:
    normalized = base_url.strip().rstrip("/")
    if not normalized:
        raise OpenAICompatibleError("OpenAI-compatible base_url is empty.")
    if normalized.endswith("/chat/completions"):
        return normalized
    parsed = urlparse(normalized)
    if parsed.path in ("", "/"):
        return normalized + "/v1/chat/completions"
    if parsed.path.endswith("/v1"):
        return normalized + "/chat/completions"
    return normalized + "/chat/completions"


def _extract_message_text(message: dict[str, Any]) -> str:
    content = message.get("content")
    if isinstance(content, str):
        return content.strip()
    if isinstance(content, list):
        parts: list[str] = []
        for part in content:
            if not isinstance(part, dict):
                continue
            text = part.get("text")
            if isinstance(text, str) and text.strip():
                parts.append(text.strip())
        return "".join(parts).strip()
    return ""


def _strip_code_fences(text: str) -> str:
    stripped = text.strip()
    if stripped.startswith("```"):
        stripped = re.sub(r"^```(?:json)?\s*", "", stripped, count=1, flags=re.IGNORECASE)
        stripped = re.sub(r"\s*```$", "", stripped, count=1)
    return stripped.strip()


def _post_json(endpoint: str, *, api_key: str, payload: dict[str, Any]) -> dict[str, Any]:
    body = json.dumps(payload).encode("utf-8")
    last_error: Exception | None = None
    for attempt in range(5):
        req = request.Request(
            endpoint,
            data=body,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        try:
            with request.urlopen(req, timeout=180) as response:
                raw = response.read().decode("utf-8", errors="replace")
        except error.HTTPError as exc:
            details = exc.read().decode("utf-8", errors="replace")
            retryable = exc.code in {429, 500, 502, 503, 504}
            if retryable and attempt < 4:
                time.sleep(min(2 ** attempt, 8))
                continue
            raise OpenAICompatibleError(
                f"OpenAI-compatible translate request failed with HTTP {exc.code}: {details or exc.reason}",
            ) from exc
        except error.URLError as exc:
            last_error = exc
            if attempt < 4:
                time.sleep(min(2 ** attempt, 8))
                continue
            raise OpenAICompatibleError(f"OpenAI-compatible translate request failed: {exc.reason}") from exc

        if not raw.strip():
            if attempt < 4:
                time.sleep(min(2 ** attempt, 8))
                continue
            raise OpenAICompatibleError("OpenAI-compatible translate response body was empty.")

        try:
            payload = json.loads(raw)
        except json.JSONDecodeError as exc:
            last_error = exc
            if attempt < 4:
                time.sleep(min(2 ** attempt, 8))
                continue
            raise
        if not isinstance(payload, dict):
            raise OpenAICompatibleError("OpenAI-compatible translate response was not a JSON object.")
        return payload

    if last_error is not None:
        raise last_error
    raise OpenAICompatibleError("OpenAI-compatible translate request failed without a response.")


def _extract_response_text(payload: dict[str, Any]) -> str:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        raise OpenAICompatibleError("OpenAI-compatible translate response did not contain choices.")
    first_choice = choices[0]
    if not isinstance(first_choice, dict):
        raise OpenAICompatibleError("OpenAI-compatible translate response choice had invalid shape.")
    message = first_choice.get("message")
    if not isinstance(message, dict):
        raise OpenAICompatibleError("OpenAI-compatible translate response did not contain a message.")
    text = _extract_message_text(message)
    if not text:
        raise OpenAICompatibleError("OpenAI-compatible translate response message content was empty.")
    return _strip_code_fences(text)


def _response_format_is_unsupported(message: str) -> bool:
    normalized = message.lower()
    return "response_format" in normalized or "json_schema" in normalized or "json_object" in normalized


def _normalize_translation_payload(parsed: Any) -> dict[str, Any]:
    if isinstance(parsed, dict):
        if isinstance(parsed.get("translations"), list):
            return parsed
        if "id" in parsed and "lines" in parsed:
            return {"translations": [parsed]}
        raise OpenAICompatibleError("OpenAI-compatible translate response JSON object did not contain translations.")
    if isinstance(parsed, list):
        return {"translations": parsed}
    raise OpenAICompatibleError("OpenAI-compatible translate response body was neither an object nor an array.")


def _merge_translation_payloads(parts: list[Any]) -> dict[str, Any]:
    merged: list[Any] = []
    for part in parts:
        normalized = _normalize_translation_payload(part)
        merged.extend(normalized["translations"])
    if not merged:
        raise OpenAICompatibleError("OpenAI-compatible translate response did not contain translations.")
    return {"translations": merged}


def _parse_translation_payload(text: str) -> dict[str, Any]:
    stripped = _strip_code_fences(text)
    try:
        return _normalize_translation_payload(json.loads(stripped))
    except json.JSONDecodeError:
        pass

    decoder = json.JSONDecoder()
    parts: list[Any] = []
    index = 0
    length = len(stripped)

    while index < length:
        while index < length and stripped[index].isspace():
            index += 1
        if index >= length:
            break

        if stripped[index] not in "[{":
            if parts:
                break
            next_object = min((pos for pos in (stripped.find("{", index), stripped.find("[", index)) if pos != -1), default=-1)
            if next_object == -1:
                raise OpenAICompatibleError("OpenAI-compatible translate response did not contain parseable JSON.")
            index = next_object

        try:
            part, end = decoder.raw_decode(stripped, index)
        except json.JSONDecodeError as exc:
            if parts:
                break
            raise OpenAICompatibleError(f"OpenAI-compatible translate response JSON parse failed: {exc}") from exc
        parts.append(part)
        index = end

    if not parts:
        raise OpenAICompatibleError("OpenAI-compatible translate response did not contain parseable JSON.")
    if len(parts) == 1:
        return _normalize_translation_payload(parts[0])
    return _merge_translation_payloads(parts)


def generate_json_response(*, base_url: str, api_key: str, model: str, prompt: str, schema: dict[str, Any]) -> dict[str, Any]:
    del schema
    endpoint = _normalize_endpoint(base_url)
    base_payload: dict[str, Any] = {
        "model": model,
        "messages": [
            {"role": "system", "content": "You translate subtitle cues and must return valid JSON only."},
            {"role": "user", "content": prompt},
        ],
        "temperature": 0.2,
    }

    attempts = [
        {"response_format": {"type": "json_object"}},
        {},
    ]
    last_error: Exception | None = None
    for extra in attempts:
        try:
            response_payload = _post_json(endpoint, api_key=api_key, payload={**base_payload, **extra})
            return _parse_translation_payload(_extract_response_text(response_payload))
        except Exception as exc:
            last_error = exc
            if extra and _response_format_is_unsupported(str(exc)):
                continue
            if not extra:
                raise
    assert last_error is not None
    raise last_error
