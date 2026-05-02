# noveltts

**noveltts** converts novel and book text files to speech audio files using
[gTTS](https://github.com/pndurette/gTTS) (Google Text-to-Speech).

## Features

- Automatically detects chapter/part headings and processes each section
- Splits long texts into gTTS-compatible chunks at sentence boundaries
- Generates numbered MP3 files ready for playback
- Configurable language, speech rate, and chunk size
- Simple command-line interface

## Installation

```bash
pip install noveltts
```

Or from source:

```bash
git clone https://github.com/onestao/noveltts.git
cd noveltts
pip install -e .
```

## Usage

### Command line

```bash
# Convert a novel to audio files (output saved to ./output/)
noveltts my_novel.txt

# Specify output directory and language
noveltts my_novel.txt -o audio/ -l en

# French text with slow speech
noveltts roman.txt -l fr --slow

# Custom basename for output files
noveltts story.txt -b chapter --output-dir chapters/

# Show all options
noveltts --help
```

### Python API

```python
from noveltts import TTSEngine, TextProcessor

# High-level: convert a file in one call
engine = TTSEngine(lang="en", output_dir="audio/")
paths = engine.convert_file("my_novel.txt")

# Fine-grained control
processor = TextProcessor(chunk_size=3000)
text = processor.load("my_novel.txt")
chunks = processor.process(text)          # clean + split chapters + chunk

engine = TTSEngine(lang="en", output_dir="audio/")
engine.convert_chunks(chunks, basename="novel")
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `INPUT` | — | Path to input text file |
| `-o / --output-dir` | `output/` | Directory for generated MP3 files |
| `-b / --basename` | file stem | Filename prefix for output files |
| `-l / --lang` | `en` | BCP-47 language code |
| `--slow` | off | Use slow speech rate |
| `--chunk-size N` | `4000` | Max characters per TTS request |
| `-v / --verbose` | off | Enable verbose logging |

## Output

Files are named `<basename>_NNNN.mp3` (e.g. `novel_0001.mp3`, `novel_0002.mp3`, …).

## Running tests

```bash
pip install -e ".[dev]"
pytest
```

## License

MIT
