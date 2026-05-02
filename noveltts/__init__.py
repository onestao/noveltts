"""
noveltts - Convert novel/book text files to speech audio files.
"""

__version__ = "0.1.0"
__author__ = "noveltts contributors"

from .engine import TTSEngine
from .processor import TextProcessor

__all__ = ["TTSEngine", "TextProcessor"]
