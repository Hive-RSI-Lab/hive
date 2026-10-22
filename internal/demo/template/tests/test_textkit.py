import unittest

import textkit


class TestTextkit(unittest.TestCase):
    def test_word_count(self):
        self.assertEqual(textkit.word_count("a b  c"), 3)
        self.assertEqual(textkit.word_count(""), 0)

    def test_reverse_words(self):
        self.assertEqual(textkit.reverse_words("one two three"), "three two one")


if __name__ == "__main__":
    unittest.main()
