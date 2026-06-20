from __future__ import annotations

import unittest

from subflow.subtitle_io import SubtitleCue
from subflow.translate_job import TranslateJobRequest, _build_prompt, _build_repair_prompt, _needs_untranslated_repair, _postprocess_translation_text


def make_cue(text: str) -> SubtitleCue:
    return SubtitleCue(index=1, start="00:00:01,000", end="00:00:02,000", text=text)


class TranslateJobRepairTests(unittest.TestCase):
    def test_bare_speaker_label_still_needs_repair(self) -> None:
        cue = make_cue("[Matty]")
        self.assertTrue(_needs_untranslated_repair("[Matty]", cue))

    def test_mixed_language_speaker_label_still_needs_repair(self) -> None:
        cue = make_cue("[Rya] 我去了他住的那家老旅馆")
        self.assertTrue(_needs_untranslated_repair("[Rya] 我去了他住的那家老旅馆", cue))

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

    def test_acronym_with_dialogue_punctuation_still_needs_repair(self) -> None:
        cue = make_cue("NASA?")
        self.assertTrue(_needs_untranslated_repair("NASA？", cue))

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
        self.assertIn("NASA?, RPM., 1G., or 2G", prompt)

    def test_repair_prompt_prefers_chinese_rendering_for_recurring_named_entities(self) -> None:
        cue = make_cue("Translucent.")
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        prompt = _build_repair_prompt(request, [(cue, "Translucent。")])
        self.assertIn("Tom, Murph, Hughie, Translucent, Gargantua, Lazarus, TARS, or CASE", prompt)
        self.assertIn("Chinese viewers normally expect that exact form", prompt)
        self.assertIn("NASA, GPS, FBI, CIA, USB, AI, and RPM", prompt)
        self.assertIn("NASA?, RPM., 1G., 2G, or Plan B", prompt)

    def test_postprocess_normalizes_known_named_entities(self) -> None:
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        text = "TARS，后退，请。\nCASE！\nCompound V。\nA-Train 来了。\nM.M.，你先上。"
        got = _postprocess_translation_text(text, request)
        self.assertIn("塔斯，后退，请。", got)
        self.assertIn("凯斯！", got)
        self.assertIn("五号化合物。", got)
        self.assertIn("火车头 来了。", got)
        self.assertIn("母乳，你先上。", got)

    def test_postprocess_keeps_allowed_acronyms_raw(self) -> None:
        request = TranslateJobRequest(
            input_path=__file__,
            output_path=__file__,
            target_language="Chinese",
        )
        got = _postprocess_translation_text("我们是NASA。GPS 失灵了。", request)
        self.assertIn("NASA", got)
        self.assertIn("GPS", got)


if __name__ == "__main__":
    unittest.main()
