#!usr/bin/env python3

import sys
from pathlib import Path

sys.path.append(str(Path(__file__).absolute().parent))

if __name__ == '__main__':
    from taraxa import perf_test

    perf_test.run()
