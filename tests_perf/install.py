#!usr/bin/env python3

import shutil
import os
import sys
from pathlib import Path

sys.path.append(str(Path(__file__).parent.absolute()))

if __name__ == '__main__':
    from paths import *
    from taraxa.util import call

    call('pip install --upgrade pip', cwd=base_dir)
    call('pip install -r requirements.txt', cwd=base_dir)
    try:
        import plyvel
    except ModuleNotFoundError:
        os.makedirs(deps_dir, exist_ok=True)
        call("git clone https://github.com/wbolster/plyvel", cwd=deps_dir)
        plyvel_dir = deps_dir.joinpath('plyvel')
        call("cython --cplus --fast-fail --annotate plyvel/_plyvel.pyx", cwd=plyvel_dir)
        call("python setup.py build_ext --inplace --force", cwd=plyvel_dir)
        call("python setup.py install", cwd=plyvel_dir)
    finally:
        shutil.rmtree(deps_dir, ignore_errors=True)
