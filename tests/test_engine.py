"""Tests for noveltts.engine (gTTS calls are mocked)."""

import pytest
from pathlib import Path
from unittest.mock import MagicMock, patch, call

from noveltts.engine import TTSEngine


@pytest.fixture
def engine(tmp_path):
    return TTSEngine(lang="en", slow=False, output_dir=str(tmp_path))


class TestTTSEngine:
    def test_output_dir_created(self, tmp_path):
        out = tmp_path / "new_dir"
        assert not out.exists()
        TTSEngine(output_dir=str(out))
        assert out.is_dir()

    @patch("noveltts.engine.gTTS", create=True)
    def test_convert_chunk_saves_file(self, mock_gtts_cls, engine, tmp_path):
        mock_tts = MagicMock()
        mock_gtts_cls.return_value = mock_tts

        # Patch import inside engine module
        with patch.dict("sys.modules", {"gtts": MagicMock(gTTS=mock_gtts_cls)}):
            with patch("noveltts.engine.TTSEngine.convert_chunk") as mock_cc:
                mock_cc.return_value = tmp_path / "out_0001.mp3"
                path = engine.convert_chunk("Hello", "out_0001.mp3")
                mock_cc.assert_called_once_with("Hello", "out_0001.mp3")

    def test_convert_chunks_returns_paths(self, engine, tmp_path):
        with patch.object(engine, "convert_chunk") as mock_cc:
            mock_cc.side_effect = lambda text, fname: tmp_path / fname
            paths = engine.convert_chunks(["chunk1", "chunk2"], basename="book")
            assert len(paths) == 2
            assert all(isinstance(p, Path) for p in paths)
            # filenames should be numbered
            names = [p.name for p in paths]
            assert "book_0001.mp3" in names
            assert "book_0002.mp3" in names

    def test_convert_chunks_start_index(self, engine, tmp_path):
        with patch.object(engine, "convert_chunk") as mock_cc:
            mock_cc.side_effect = lambda text, fname: tmp_path / fname
            paths = engine.convert_chunks(["chunk1"], basename="book", start_index=4)
            assert paths[0].name == "book_0005.mp3"

    def test_convert_file_uses_processor(self, engine, tmp_path):
        src = tmp_path / "novel.txt"
        src.write_text("Chapter 1\nHello world.", encoding="utf-8")

        mock_processor = MagicMock()
        mock_processor.load.return_value = "Chapter 1\nHello world."
        mock_processor.process.return_value = ["Chapter 1\nHello world."]

        with patch.object(engine, "convert_chunks") as mock_cc:
            mock_cc.return_value = [tmp_path / "novel_0001.mp3"]
            paths = engine.convert_file(str(src), processor=mock_processor)

        mock_processor.load.assert_called_once_with(str(src))
        mock_processor.process.assert_called_once()
        assert len(paths) == 1

    def test_convert_file_default_basename(self, engine, tmp_path):
        src = tmp_path / "my_novel.txt"
        src.write_text("Some text.", encoding="utf-8")

        mock_processor = MagicMock()
        mock_processor.load.return_value = "Some text."
        mock_processor.process.return_value = ["Some text."]

        with patch.object(engine, "convert_chunks") as mock_cc:
            mock_cc.return_value = []
            engine.convert_file(str(src), processor=mock_processor)
            # basename should default to file stem
            mock_cc.assert_called_once_with(["Some text."], basename="my_novel")
