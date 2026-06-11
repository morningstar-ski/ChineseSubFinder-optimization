from __future__ import annotations

import json
import os
import subprocess
import tempfile
from pathlib import Path
from typing import Any

from .config import SubflowConfig, load_subflow_config


class GeminiUnavailableError(RuntimeError):
    pass


def _import_genai():
    try:
        from google import genai
    except Exception as exc:  # pragma: no cover - depends on local install
        raise GeminiUnavailableError(
            "google-genai is not installed. Run `python -m pip install -e .` first.",
        ) from exc
    return genai


def create_client(config: SubflowConfig | None = None):
    genai = _import_genai()
    settings = config or load_subflow_config()
    if not settings.api_key:
        raise GeminiUnavailableError(
            "Set GEMINI_API_KEY or configure api_key in subflow config before running a Gemini-backed subflow command.",
        )
    return genai.Client(api_key=settings.api_key)


def upload_audio_for_gemini(client: Any, source: Path, work_dir: Path | None = None) -> tuple[Any, Path]:
    temp_dir = work_dir or Path(tempfile.mkdtemp(prefix="subflow-audio-"))
    if source.suffix.lower() in {".wav", ".flac", ".mp3", ".m4a", ".aac", ".ogg", ".opus"}:
        uploaded = client.files.upload(file=str(source))
        return uploaded, source

    target = temp_dir / f"{source.stem}.flac"
    ffmpeg = os.getenv("FFMPEG_PATH") or "ffmpeg"
    command = [
        ffmpeg,
        "-y",
        "-i",
        str(source),
        "-vn",
        "-ac",
        "1",
        "-ar",
        "16000",
        "-c:a",
        "flac",
        str(target),
    ]
    subprocess.run(command, check=True, text=True, capture_output=True)
    uploaded = client.files.upload(file=str(target))
    return uploaded, target


def generate_json_response(client: Any, *, model: str, contents: Any, schema: dict[str, Any]) -> dict[str, Any]:
    response = client.models.generate_content(
        model=model,
        contents=contents,
        config={
            "response_mime_type": "application/json",
            "response_json_schema": schema,
        },
    )
    text = getattr(response, "text", None)
    if not text:
        raise RuntimeError("Gemini returned an empty response.")
    payload = json.loads(text)
    if not isinstance(payload, dict):
        raise RuntimeError("Gemini response was not a JSON object.")
    return payload
