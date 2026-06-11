from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class SubtitleCue:
    index: int
    start: float
    end: float
    text: str


_TIMESTAMP_RE = re.compile(
    r"^(?P<h>\d{2}):(?P<m>\d{2}):(?P<s>\d{2})[,.:](?P<ms>\d{3})$",
)


def parse_timestamp(value: str) -> float:
    match = _TIMESTAMP_RE.match(value.strip())
    if not match:
        raise ValueError(f"Invalid subtitle timestamp: {value!r}")
    hours = int(match.group("h"))
    minutes = int(match.group("m"))
    seconds = int(match.group("s"))
    milliseconds = int(match.group("ms"))
    return hours * 3600 + minutes * 60 + seconds + milliseconds / 1000


def format_timestamp(value: float) -> str:
    if value < 0:
        value = 0.0
    total_ms = int(round(value * 1000))
    hours, remainder = divmod(total_ms, 3600_000)
    minutes, remainder = divmod(remainder, 60_000)
    seconds, milliseconds = divmod(remainder, 1000)
    return f"{hours:02d}:{minutes:02d}:{seconds:02d},{milliseconds:03d}"


def parse_srt_text(text: str) -> list[SubtitleCue]:
    normalized = text.replace("\ufeff", "").replace("\r\n", "\n").replace("\r", "\n").strip()
    if not normalized:
        return []

    cues: list[SubtitleCue] = []
    for block in re.split(r"\n{2,}", normalized):
        lines = [line.rstrip() for line in block.split("\n") if line.strip() or line == ""]
        if not lines:
            continue

        time_line_index = 0
        if len(lines) >= 2 and lines[0].strip().isdigit() and "-->" in lines[1]:
            time_line_index = 1
        if "-->" not in lines[time_line_index]:
            raise ValueError(f"Invalid SRT block: {block!r}")

        start_text, end_text = [item.strip() for item in lines[time_line_index].split("-->", 1)]
        cue_text = "\n".join(lines[time_line_index + 1 :]).strip()
        cues.append(
            SubtitleCue(
                index=len(cues) + 1,
                start=parse_timestamp(start_text),
                end=parse_timestamp(end_text),
                text=cue_text,
            ),
        )

    return cues


def read_srt(path: Path) -> list[SubtitleCue]:
    return parse_srt_text(path.read_text(encoding="utf-8-sig"))


def write_srt(cues: list[SubtitleCue]) -> str:
    parts: list[str] = []
    for index, cue in enumerate(cues, start=1):
        parts.append(str(index))
        parts.append(f"{format_timestamp(cue.start)} --> {format_timestamp(cue.end)}")
        parts.extend(cue.text.rstrip().splitlines() or [""])
        parts.append("")
    return "\n".join(parts).rstrip() + "\n"


def display_width(text: str) -> int:
    total = 0
    for char in text:
        if char in {"\n", "\r"}:
            continue
        if unicodedata.combining(char):
            continue
        total += 2 if unicodedata.east_asian_width(char) in {"F", "W"} else 1
    return total
