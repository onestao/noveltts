"""Allow ``python -m noveltts`` invocation."""

import sys
from .cli import main

sys.exit(main())
