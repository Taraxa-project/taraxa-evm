#!usr/bin/env python3

import sys
from os import path

sys.path.append(path.dirname(path.abspath(__file__)))

if __name__ == '__main__':
    import cli

    cli.ENTRY_POINT()
