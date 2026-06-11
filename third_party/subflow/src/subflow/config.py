from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

try:  # Python 3.11+
    import tomllib
except ModuleNotFoundError:  # pragma: no cover - Python 3.10 fallback
    tomllib = None  # type: ignore[assignment]


@dataclass(frozen=True)
class SubflowConfig:
    api_key: str | None
    api_key_source: str | None
    google_credentials_path: Path | None
    google_project_id: str | None
    google_stt_bucket: str | None
    google_stt_language: str
    google_stt_model: str
    asr_model: str
    translate_model: str
    translate_style: str | None
    qa_model: str
    config_path: Path | None


def default_config_candidates() -> list[Path]:
    candidates: list[Path] = []
    appdata = os.getenv("APPDATA")
    if appdata:
        candidates.append(Path(appdata) / "subflow" / "config.toml")
    candidates.append(Path.home() / ".config" / "subflow" / "config.toml")
    return candidates


def _load_toml(path: Path) -> dict[str, Any]:
    if tomllib is None:  # pragma: no cover - defensive fallback
        return {}
    with path.open("rb") as handle:
        data = tomllib.load(handle)
    if isinstance(data, dict):
        return data
    return {}


def _first_string(*values: Any, default: str) -> str:
    for value in values:
        if isinstance(value, str) and value.strip():
            return value.strip()
    return default


def load_subflow_config(path: Path | None = None) -> SubflowConfig:
    config_path = path
    if config_path is None:
        env_path = os.getenv("SUBFLOW_CONFIG")
        if env_path:
            config_path = Path(env_path)
        else:
            for candidate in default_config_candidates():
                if candidate.exists():
                    config_path = candidate
                    break

    data: dict[str, Any] = {}
    if config_path is not None and config_path.exists():
        data = _load_toml(config_path)

    gemini = data.get("gemini", {}) if isinstance(data.get("gemini", {}), dict) else {}
    subflow = data.get("subflow", {}) if isinstance(data.get("subflow", {}), dict) else {}

    api_key_source: str | None = None
    api_key = os.getenv("GEMINI_API_KEY")
    if api_key:
        api_key_source = "GEMINI_API_KEY"
    else:
        api_key = _first_string(
            gemini.get("api_key") if isinstance(gemini, dict) else None,
            subflow.get("api_key") if isinstance(subflow, dict) else None,
            data.get("api_key"),
            default="",
        ) or None
        if api_key:
            api_key_source = "config"

    google_credentials_path: Path | None = None
    credentials_env = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
    if isinstance(credentials_env, str) and credentials_env.strip():
        candidate = Path(credentials_env.strip()).expanduser()
        if candidate.exists():
            google_credentials_path = candidate
    if google_credentials_path is None:
        credentials_value = _first_string(
            subflow.get("google_application_credentials") if isinstance(subflow, dict) else None,
            gemini.get("google_application_credentials") if isinstance(gemini, dict) else None,
            data.get("google_application_credentials"),
            default="",
        )
        if credentials_value:
            candidate = Path(credentials_value).expanduser()
            if candidate.exists():
                google_credentials_path = candidate

    google_project_id = _first_string(
        os.getenv("GOOGLE_CLOUD_PROJECT"),
        os.getenv("GCLOUD_PROJECT"),
        subflow.get("google_cloud_project") if isinstance(subflow, dict) else None,
        gemini.get("google_cloud_project") if isinstance(gemini, dict) else None,
        data.get("google_cloud_project"),
        default="",
    ) or None
    if google_project_id is None and google_credentials_path is not None and google_credentials_path.exists():
        try:
            payload = json.loads(google_credentials_path.read_text(encoding="utf-8"))
            if isinstance(payload, dict):
                project_id_value = payload.get("project_id")
                if isinstance(project_id_value, str) and project_id_value.strip():
                    google_project_id = project_id_value.strip()
        except Exception:
            pass

    google_stt_bucket = _first_string(
        os.getenv("SUBFLOW_GOOGLE_STT_BUCKET"),
        subflow.get("google_stt_bucket") if isinstance(subflow, dict) else None,
        gemini.get("google_stt_bucket") if isinstance(gemini, dict) else None,
        data.get("google_stt_bucket"),
        default="",
    ) or None

    google_stt_language = _first_string(
        os.getenv("SUBFLOW_GOOGLE_STT_LANGUAGE"),
        subflow.get("google_stt_language") if isinstance(subflow, dict) else None,
        gemini.get("google_stt_language") if isinstance(gemini, dict) else None,
        data.get("google_stt_language"),
        default="en-US",
    )
    google_stt_model = _first_string(
        os.getenv("SUBFLOW_GOOGLE_STT_MODEL"),
        subflow.get("google_stt_model") if isinstance(subflow, dict) else None,
        gemini.get("google_stt_model") if isinstance(gemini, dict) else None,
        data.get("google_stt_model"),
        default="latest_long",
    )

    def _resolve_model(env_name: str, config_key: str, default: str) -> str:
        value = os.getenv(env_name)
        if isinstance(value, str) and value.strip():
            return value.strip()
        return _first_string(
            subflow.get(config_key) if isinstance(subflow, dict) else None,
            gemini.get(config_key) if isinstance(gemini, dict) else None,
            data.get(config_key),
            default=default,
        )

    return SubflowConfig(
        api_key=api_key,
        api_key_source=api_key_source,
        google_credentials_path=google_credentials_path,
        google_project_id=google_project_id,
        google_stt_bucket=google_stt_bucket,
        google_stt_language=google_stt_language,
        google_stt_model=google_stt_model,
        asr_model=_resolve_model("SUBFLOW_ASR_MODEL", "asr_model", "gemini-2.5-flash"),
        translate_model=_resolve_model("SUBFLOW_TRANSLATE_MODEL", "translate_model", "gemini-2.5-flash"),
        translate_style=_first_string(
            os.getenv("SUBFLOW_TRANSLATE_STYLE"),
            subflow.get("translate_style") if isinstance(subflow, dict) else None,
            gemini.get("translate_style") if isinstance(gemini, dict) else None,
            data.get("translate_style"),
            default="",
        ) or None,
        qa_model=_resolve_model("SUBFLOW_QA_MODEL", "qa_model", "gemini-2.5-flash"),
        config_path=config_path if config_path is not None and config_path.exists() else None,
    )
