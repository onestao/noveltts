"""Tests for noveltts.processor."""

import pytest
from noveltts.processor import TextProcessor, DEFAULT_CHUNK_SIZE


# ---------------------------------------------------------------------------
# TextProcessor.clean
# ---------------------------------------------------------------------------

class TestClean:
    def setup_method(self):
        self.proc = TextProcessor()

    def test_collapses_blank_lines(self):
        text = "Hello\n\n\n\nWorld"
        result = self.proc.clean(text)
        assert "\n\n\n" not in result
        assert "Hello" in result
        assert "World" in result

    def test_strips_leading_trailing_whitespace(self):
        result = self.proc.clean("  Hello World  ")
        assert result == "Hello World"

    def test_normalizes_typographic_quotes(self):
        result = self.proc.clean("\u201cHello\u201d \u2018World\u2019")
        assert '"Hello"' in result
        assert "'World'" in result

    def test_normalizes_dashes(self):
        result = self.proc.clean("Em\u2014dash and en\u2013dash")
        assert "\u2014" not in result
        assert "\u2013" not in result

    def test_empty_string(self):
        assert self.proc.clean("") == ""


# ---------------------------------------------------------------------------
# TextProcessor.split_chapters
# ---------------------------------------------------------------------------

class TestSplitChapters:
    def setup_method(self):
        self.proc = TextProcessor()

    def test_splits_on_chapter_keyword(self):
        text = "Prologue\nSome intro text.\nChapter 1\nFirst chapter.\nChapter 2\nSecond chapter."
        parts = self.proc.split_chapters(text)
        assert len(parts) == 3

    def test_splits_on_prologue(self):
        text = "Prologue\nIntro text here.\nChapter One\nMain text."
        parts = self.proc.split_chapters(text)
        assert len(parts) == 2

    def test_no_chapter_markers(self):
        text = "Just a single block of text with no chapter markers."
        parts = self.proc.split_chapters(text)
        assert len(parts) == 1
        assert parts[0] == text

    def test_empty_string(self):
        parts = self.proc.split_chapters("")
        assert len(parts) == 1


# ---------------------------------------------------------------------------
# TextProcessor.chunk
# ---------------------------------------------------------------------------

class TestChunk:
    def setup_method(self):
        self.proc = TextProcessor(chunk_size=50)

    def test_short_text_not_chunked(self):
        text = "Short text."
        chunks = self.proc.chunk(text)
        assert chunks == ["Short text."]

    def test_long_text_is_chunked(self):
        # 200 characters of text
        text = "Hello world. " * 20
        chunks = self.proc.chunk(text)
        assert len(chunks) > 1
        for chunk in chunks:
            assert len(chunk) <= 50

    def test_chunks_join_to_original_content(self):
        """All words in the original text must appear in chunks."""
        text = "The quick brown fox jumps over the lazy dog. " * 10
        chunks = self.proc.chunk(text)
        rejoined = " ".join(chunks)
        for word in ["quick", "brown", "fox", "lazy", "dog"]:
            assert word in rejoined

    def test_chunk_size_override(self):
        text = "a" * 200
        chunks = self.proc.chunk(text, size=100)
        assert len(chunks) == 2

    def test_default_chunk_size(self):
        proc = TextProcessor()
        assert proc.chunk_size == DEFAULT_CHUNK_SIZE


# ---------------------------------------------------------------------------
# TextProcessor.process
# ---------------------------------------------------------------------------

class TestProcess:
    def setup_method(self):
        self.proc = TextProcessor(chunk_size=100)

    def test_returns_list_of_strings(self):
        text = "Chapter 1\nSome text here. " * 5
        result = self.proc.process(text)
        assert isinstance(result, list)
        assert all(isinstance(s, str) for s in result)

    def test_no_empty_chunks(self):
        text = "Chapter 1\nSome text. " * 10
        result = self.proc.process(text)
        assert all(s.strip() for s in result)

    def test_load_reads_file(self, tmp_path):
        p = tmp_path / "novel.txt"
        p.write_text("Hello, world!", encoding="utf-8")
        text = self.proc.load(str(p))
        assert text == "Hello, world!"
