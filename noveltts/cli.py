"""Command-line interface for noveltts."""

import argparse
import logging
import sys
from pathlib import Path

from . import __version__
from .engine import TTSEngine
from .processor import TextProcessor


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="noveltts",
        description="Convert novel / book text files to speech audio files.",
    )
    parser.add_argument("--version", action="version", version=f"%(prog)s {__version__}")
    parser.add_argument(
        "input",
        metavar="INPUT",
        help="path to the input text file",
    )
    parser.add_argument(
        "-o",
        "--output-dir",
        default="output",
        metavar="DIR",
        help="directory for generated audio files (default: output/)",
    )
    parser.add_argument(
        "-b",
        "--basename",
        default=None,
        metavar="NAME",
        help="filename prefix for audio files (default: input file stem)",
    )
    parser.add_argument(
        "-l",
        "--lang",
        default="en",
        metavar="LANG",
        help="BCP-47 language code (default: en)",
    )
    parser.add_argument(
        "--slow",
        action="store_true",
        help="use slow speech rate",
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=4000,
        metavar="N",
        help="maximum characters per TTS chunk (default: 4000)",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="enable verbose logging",
    )
    return parser


def main(argv=None) -> int:
    """Entry point for the ``noveltts`` command."""
    parser = _build_parser()
    args = parser.parse_args(argv)

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s: %(message)s",
    )

    input_path = Path(args.input)
    if not input_path.is_file():
        print(f"Error: '{input_path}' is not a file.", file=sys.stderr)
        return 1

    processor = TextProcessor(chunk_size=args.chunk_size)
    engine = TTSEngine(lang=args.lang, slow=args.slow, output_dir=args.output_dir)

    try:
        paths = engine.convert_file(
            str(input_path),
            output_basename=args.basename,
            processor=processor,
        )
    except Exception as exc:  # pragma: no cover
        logging.error("Conversion failed: %s", exc)
        return 1

    print(f"Done. {len(paths)} file(s) written to '{args.output_dir}/'.")
    return 0


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
