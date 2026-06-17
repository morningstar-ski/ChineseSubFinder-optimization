from __future__ import annotations

import json
import re
from typing import Any
from urllib import error, request


class OpenAICompatibleError(RuntimeError):
    pass


def _normalize_endpoint(base_url: str) -> str:
    normalized = base_url.strip().rstrip("/")
    if not normalized:
        raise OpenAICompatibleError("OpenAI-compatible base_url is empty.")
    if normalized.endswith("/chat/completions"):
        return normalized
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
        raise OpenAICompatibleError(
            f"OpenAI-compatible translate request failed with HTTP {exc.code}: {details or exc.reason}",
        ) from exc
    except error.URLError as exc:
        raise OpenAICompatibleError(f"OpenAI-compatible translate request failed: {exc.reason}") from exc

    payload = json.loads(raw)
    if not isinstance(payload, dict):
        raise OpenAICompatibleError("OpenAI-compatible translate response was not a JSON object.")
    return payload


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


def _normalize_completion_result(text: str) -> dict[str, Any]:
    parsed = json.loads(text)
    if isinstance(parsed, dict):
        return parsed
    if isinstance(parsed, list):
        return {"translations": parsed}
    raise OpenAICompatibleError("OpenAI-compatible translate response was not a JSON object.")


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
            response_text = _extract_response_text(response_payload)
            result = _normalize_completion_result(response_text)
            result["__raw_api_payload"] = response_payload
            result["__raw_completion_text"] = response_text
            return result
        except Exception as exc:
            last_error = exc
            if extra and _response_format_is_unsupported(str(exc)):
                continue
            if not extra:
                raise
    assert last_error is not None
    raise last_error
