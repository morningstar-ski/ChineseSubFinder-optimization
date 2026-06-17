from __future__ import annotations

import unittest

from subflow.openai_compatible_client import OpenAICompatibleError, _normalize_completion_result


class NormalizeCompletionResultTests(unittest.TestCase):
    def test_accepts_object_payload(self) -> None:
        result = _normalize_completion_result('{"translations":[{"id":1,"lines":["你好"]}]}')
        self.assertEqual(result["translations"][0]["id"], 1)

    def test_wraps_top_level_array_as_translations(self) -> None:
        result = _normalize_completion_result('[{"id":1,"lines":["你好"]}]')
        self.assertEqual(result, {"translations": [{"id": 1, "lines": ["你好"]}]})

    def test_rejects_scalar_payload(self) -> None:
        with self.assertRaises(OpenAICompatibleError):
            _normalize_completion_result('"hello"')


if __name__ == "__main__":
    unittest.main()
