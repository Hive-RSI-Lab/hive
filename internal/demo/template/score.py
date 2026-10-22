"""Prints how many textkit functions pass their checks. Higher is better."""
import textkit

CHECKS = {
    "word_count": lambda: textkit.word_count("a b  c") == 3,
    "reverse_words": lambda: textkit.reverse_words("a b c") == "c b a",
    "capitalize_words": lambda: textkit.capitalize_words("hello big world") == "Hello Big World",
    "is_palindrome": lambda: textkit.is_palindrome("A man, a plan, a canal: Panama") and not textkit.is_palindrome("hive"),
    "char_frequency": lambda: textkit.char_frequency("aab b") == {"a": 2, "b": 2},
    "truncate": lambda: textkit.truncate("abcdefgh", 6) == "abc..." and textkit.truncate("abc", 6) == "abc",
    "slugify": lambda: textkit.slugify("Hello, World! 2") == "hello-world-2",
    "wrap": lambda: textkit.wrap("the quick brown fox", 10) == ["the quick", "brown fox"],
}

passed = 0
for name, check in CHECKS.items():
    try:
        ok = bool(check())
    except Exception:
        ok = False
    print(("pass " if ok else "fail ") + name)
    passed += ok
print(passed)
