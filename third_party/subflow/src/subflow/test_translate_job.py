from __future__ import annotations

import unittest

from subflow.subtitle_io import SubtitleCue
from subflow.translate_job import (
    CueChunk,
    TranslateJobRequest,
    _build_prompt,
    _build_repair_prompt,
    _chunk_cues,
    _needs_untranslated_repair,
    _postprocess_translation_text,
)


def make_cue(text: str, *, index: int = 1) -> SubtitleCue:
    return SubtitleCue(index=index, start="00:00:01,000", end="00:00:02,000", text=text)


class TranslateJobRepairTests(unittest.TestCase):
    def test_bare_speaker_label_still_needs_repair(self) -> None:
        cue = make_cue("[Matty]")
        self.assertTrue(_needs_untranslated_repair("[Matty]", cue))

    def test_short_acronym_can_remain_english(self) -> None:
        cue = make_cue("FBI")
        self.assertFalse(_needs_untranslated_repair("FBI", cue))

    def test_single_letter_fragment_needs_repair(self) -> None:
        cue = make_cue("S.")
        self.assertTrue(_needs_untranslated_repair("S.", cue))

    def test_untranslated_dialogue_still_needs_repair(self) -> None:
        cue = make_cue("I need your help.")
        self.assertTrue(_needs_untranslated_repair("I need your help.", cue))

    def test_short_dashed_dialogue_still_needs_repair(self) -> None:
        cue = make_cue("- In secret.")
        self.assertTrue(_needs_untranslated_repair("- In secret.", cue))

    def test_mixed_script_dialogue_needs_repair(self) -> None:
        cue = make_cue("Hello, Summer.")
        self.assertTrue(_needs_untranslated_repair("你好，Summer。", cue))

    def test_translate_prompt_prefers_chinese_rendering_for_recurring_named_entities(self) -> None:
        cue = make_cue("Hughie, TARS, and Gargantua.")
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        prompt = _build_prompt(request, [cue])
        self.assertIn("Tom, Murph, Hughie, Translucent, Gargantua, Lazarus, TARS, or CASE", prompt)
        self.assertIn("standard Chinese transliteration or conventional Chinese rendering", prompt)
        self.assertIn("NASA, GPS, FBI, CIA, USB, AI, and RPM", prompt)
        self.assertIn("context_only", prompt)
        self.assertIn("Do not return items for context_only cues", prompt)

    def test_repair_prompt_uses_neighbor_context_without_allowing_drift(self) -> None:
        cue = make_cue("Translucent.")
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        prompt = _build_repair_prompt(request, [(cue, "Translucent。")])
        self.assertIn("previous_source_lines", prompt)
        self.assertIn("next_source_lines", prompt)
        self.assertIn("Do not borrow full meaning from neighboring cues", prompt)
        self.assertIn("NASA, GPS, FBI, CIA, USB, AI, and RPM", prompt)

    def test_postprocess_removes_slash_punctuation_artifact(self) -> None:
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        got = _postprocess_translation_text("刚刚毁了我们GTA的评分 / 。", request)
        self.assertEqual("刚刚毁了我们GTA的评分。", got)

    def test_postprocess_keeps_allowed_acronyms_raw(self) -> None:
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        got = _postprocess_translation_text("我们的 NASA GPS 失灵了。", request)
        self.assertIn("NASA", got)
        self.assertIn("GPS", got)

    def test_chunk_builder_adds_context_window(self) -> None:
        cues = [make_cue(f"line {i + 1}", index=i + 1) for i in range(5)]
        chunks = _chunk_cues(cues, max_items=2, max_chars=9999)
        self.assertEqual(3, len(chunks))
        self.assertIsInstance(chunks[0], CueChunk)
        self.assertEqual([1, 2], [cue.index for cue in chunks[0].target_cues])
        self.assertEqual([3, 4], [cue.index for cue in chunks[0].context_after])
        self.assertEqual([1, 2], [cue.index for cue in chunks[1].context_before])
        self.assertEqual([3, 4], [cue.index for cue in chunks[1].target_cues])


if __name__ == "__main__":
    unittest.main()
