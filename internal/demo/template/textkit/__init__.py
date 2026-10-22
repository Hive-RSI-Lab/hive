"""textkit: small text utilities."""


def word_count(text):
    """Number of whitespace-separated words."""
    return len(text.split())


def reverse_words(text):
    """The words of text in reverse order, joined by single spaces."""
    return " ".join(reversed(text.split()))


def capitalize_words(text):
    """Each word with its first letter upper-cased, joined by single spaces."""
    raise NotImplementedError


def is_palindrome(text):
    """True if text reads the same backwards, ignoring case, spaces and punctuation."""
    raise NotImplementedError


def char_frequency(text):
    """A dict of character -> count, ignoring spaces."""
    raise NotImplementedError


def truncate(text, width):
    """text cut to at most width characters, ending in '...' when cut (width >= 3)."""
    raise NotImplementedError


def slugify(text):
    """Lower-case words joined by '-', keeping only letters and digits."""
    raise NotImplementedError


def wrap(text, width):
    """text broken into lines of at most width characters, on word boundaries."""
    raise NotImplementedError
