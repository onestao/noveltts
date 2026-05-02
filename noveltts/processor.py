"""Text processing utilities for noveltts."""

import re
import unicodedata
from typing import List, Optional


# Patterns that typically mark chapter headings
_CHAPTER_PATTERNS = [
    re.compile(r"^\s*chapter\s+\w+", re.IGNORECASE | re.MULTILINE),
    re.compile(r"^\s*ch\.\s*\d+", re.IGNORECASE | re.MULTILINE),
    re.compile(r"^\s*part\s+\w+", re.IGNORECASE | re.MULTILINE),
    re.compile(r"^\s*book\s+\w+", re.IGNORECASE | re.MULTILINE),
    re.compile(r"^\s*prologue\b", re.IGNORECASE | re.MULTILINE),
    re.compile(r"^\s*epilogue\b", re.IGNORECASE | re.MULTILINE),
]

# Combined alternation used for splitting
_SPLIT_PATTERN = re.compile(
    r"(?=^\s*(?:chapter\s+\w+|ch\.\s*\d+|part\s+\w+|book\s+\w+|prologue|epilogue)\b)",
    re.IGNORECASE | re.MULTILINE,
)

# Maximum characters that gTTS accepts per request (≈ 5 000, stay conservative)
DEFAULT_CHUNK_SIZE = 4000


class TextProcessor:
    """Cleans and splits novel text for TTS conversion.

    Parameters
    ----------
    chunk_size:
        Maximum number of characters per TTS chunk.  The processor
        tries to split on sentence boundaries before this limit.
    """

    def __init__(self, chunk_size: int = DEFAULT_CHUNK_SIZE) -> None:
        self.chunk_size = chunk_size

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def load(self, path: str, encoding: str = "utf-8") -> str:
        """Read a text file and return its contents."""
        with open(path, encoding=encoding, errors="replace") as fh:
            return fh.read()

    def clean(self, text: str) -> str:
        """Normalise whitespace and remove control characters."""
        # Normalize unicode
        text = unicodedata.normalize("NFC", text)
        # Replace common typographic quotes / dashes with ASCII equivalents
        text = text.replace("\u2018", "'").replace("\u2019", "'")
        text = text.replace("\u201c", '"').replace("\u201d", '"')
        text = text.replace("\u2013", "-").replace("\u2014", "--")
        # Strip non-printable control characters (keep newlines / tabs)
        text = re.sub(r"[^\S\n\t ]+", " ", text)
        # Collapse multiple blank lines into two
        text = re.sub(r"\n{3,}", "\n\n", text)
        return text.strip()

    def split_chapters(self, text: str) -> List[str]:
        """Split *text* into chapters based on common heading patterns.

        If no chapter markers are found the entire text is returned as a
        single-element list.
        """
        parts = _SPLIT_PATTERN.split(text)
        chapters = [p.strip() for p in parts if p.strip()]
        return chapters if chapters else [text]

    def chunk(self, text: str, size: Optional[int] = None) -> List[str]:
        """Break *text* into chunks no larger than *size* characters.

        Splitting is attempted at sentence boundaries (``. ``, ``! ``,
        ``? ``) before falling back to word boundaries or hard truncation.
        """
        size = size or self.chunk_size
        if len(text) <= size:
            return [text]

        chunks: List[str] = []
        remaining = text
        while remaining:
            if len(remaining) <= size:
                chunks.append(remaining)
                break
            # Look for a sentence boundary within the window
            window = remaining[:size]
            cut = self._sentence_boundary(window)
            if cut == -1:
                # Fall back to last whitespace
                cut = window.rfind(" ")
            if cut <= 0:
                # Hard cut
                cut = size
            chunks.append(remaining[:cut].rstrip())
            remaining = remaining[cut:].lstrip()
        return chunks

    def process(self, text: str) -> List[str]:
        """Full pipeline: clean → split chapters → chunk each chapter."""
        cleaned = self.clean(text)
        chapters = self.split_chapters(cleaned)
        chunks: List[str] = []
        for chapter in chapters:
            chunks.extend(self.chunk(chapter))
        return chunks

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _sentence_boundary(text: str) -> int:
        """Return the index *after* the last sentence-ending punctuation in *text*."""
        best = -1
        for m in re.finditer(r"[.!?]['\"]?\s+", text):
            best = m.end()
        return best
