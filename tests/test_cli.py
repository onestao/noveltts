"""Tests for noveltts.cli."""

import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock

from noveltts.cli import main, _build_parser


class TestBuildParser:
    def test_input_required(self):
        parser = _build_parser()
        with pytest.raises(SystemExit):
            parser.parse_args([])

    def test_defaults(self):
        parser = _build_parser()
        args = parser.parse_args(["mybook.txt"])
        assert args.input == "mybook.txt"
        assert args.output_dir == "output"
        assert args.lang == "en"
        assert args.slow is False
        assert args.chunk_size == 4000
        assert args.verbose is False
        assert args.basename is None

    def test_all_flags(self):
        parser = _build_parser()
        args = parser.parse_args([
            "book.txt", "-o", "out", "-b", "mybook",
            "-l", "fr", "--slow", "--chunk-size", "2000", "-v",
        ])
        assert args.output_dir == "out"
        assert args.basename == "mybook"
        assert args.lang == "fr"
        assert args.slow is True
        assert args.chunk_size == 2000
        assert args.verbose is True


class TestMain:
    def test_missing_file_returns_1(self, tmp_path):
        result = main([str(tmp_path / "nonexistent.txt")])
        assert result == 1

    def test_converts_file(self, tmp_path):
        src = tmp_path / "novel.txt"
        src.write_text("Chapter 1\nHello world.", encoding="utf-8")

        with patch("noveltts.cli.TTSEngine") as MockEngine:
            mock_engine_instance = MagicMock()
            mock_engine_instance.convert_file.return_value = [tmp_path / "novel_0001.mp3"]
            MockEngine.return_value = mock_engine_instance

            result = main([str(src), "-o", str(tmp_path)])

        assert result == 0
        mock_engine_instance.convert_file.assert_called_once()

    def test_version_exits(self):
        with pytest.raises(SystemExit) as exc_info:
            main(["--version"])
        assert exc_info.value.code == 0
