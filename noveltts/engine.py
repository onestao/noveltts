"""TTS engine wrapper for noveltts."""

import os
import logging
from pathlib import Path
from typing import List, Optional

logger = logging.getLogger(__name__)


class TTSEngine:
    """Converts text chunks to audio files.

    Parameters
    ----------
    lang:
        BCP-47 language tag (default ``"en"``).
    slow:
        Ask the TTS service to speak slowly (default ``False``).
    output_dir:
        Directory where audio files will be written.  Created if it does
        not exist.
    """

    def __init__(
        self,
        lang: str = "en",
        slow: bool = False,
        output_dir: str = "output",
    ) -> None:
        self.lang = lang
        self.slow = slow
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def convert_chunk(self, text: str, filename: str) -> Path:
        """Convert a single *text* chunk and save it as *filename*.

        Returns the path to the created audio file.
        """
        try:
            from gtts import gTTS  # imported lazily so tests can mock easily
        except ImportError as exc:  # pragma: no cover
            raise RuntimeError(
                "gTTS is required for audio conversion. "
                "Install it with: pip install gTTS"
            ) from exc

        dest = self.output_dir / filename
        tts = gTTS(text=text, lang=self.lang, slow=self.slow)
        tts.save(str(dest))
        logger.info("Saved %s", dest)
        return dest

    def convert_chunks(
        self,
        chunks: List[str],
        basename: str = "noveltts",
        *,
        start_index: int = 0,
    ) -> List[Path]:
        """Convert a list of text *chunks* to numbered audio files.

        Parameters
        ----------
        chunks:
            Ordered list of text snippets to synthesise.
        basename:
            Prefix for generated filenames (``<basename>_001.mp3``, …).
        start_index:
            Numbering offset (useful when resuming a long conversion).

        Returns a list of paths to the created files.
        """
        paths: List[Path] = []
        total = len(chunks)
        for i, chunk in enumerate(chunks, start=start_index + 1):
            filename = f"{basename}_{i:04d}.mp3"
            logger.info("Converting chunk %d / %d …", i - start_index, total)
            path = self.convert_chunk(chunk, filename)
            paths.append(path)
        return paths

    def convert_file(
        self,
        input_path: str,
        output_basename: Optional[str] = None,
        processor=None,
    ) -> List[Path]:
        """High-level helper: read *input_path*, process text, and convert.

        Parameters
        ----------
        input_path:
            Path to the source text file.
        output_basename:
            Stem for output filenames.  Defaults to the input file stem.
        processor:
            A :class:`~noveltts.processor.TextProcessor` instance.
            Created with default settings if *None*.
        """
        from .processor import TextProcessor

        if processor is None:
            processor = TextProcessor()

        if output_basename is None:
            output_basename = Path(input_path).stem

        text = processor.load(input_path)
        chunks = processor.process(text)
        logger.info(
            "Processing '%s' → %d chunk(s) → '%s/'",
            input_path,
            len(chunks),
            self.output_dir,
        )
        return self.convert_chunks(chunks, basename=output_basename)
