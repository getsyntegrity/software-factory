"""factory/config.json, read and checked once for every factory script."""

import json
import sys
from pathlib import Path

PATH = Path(__file__).resolve().parents[1] / "config.json"


def load(prog):
    """Return the config, or exit with `<prog>: <why>` when it can't be used.

    base_branch has no fallback: a factory that guessed it could build jobs
    from, and merge them into, the wrong branch.
    """
    config = json.loads(PATH.read_text())
    base = config.get("base_branch")
    if not isinstance(base, str) or not base.strip():
        sys.exit(f'{prog}: {PATH.parent.name}/{PATH.name} needs "base_branch": '
                 "the branch jobs build on and merge into")
    return config
