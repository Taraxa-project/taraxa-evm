#!usr/bin/env python3

import sys
from pathlib import Path

sys.path.append(str(Path(__file__).absolute().parent.parent))

if __name__ == '__main__':
    from apps import shell
    from apps.compute_state_db import *
    from apps.blocks_transactions import *

    shell.run_cli()
