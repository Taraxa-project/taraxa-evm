#!usr/bin/env python3

from taraxa.util import call
import shutil

from paths import *

call("rm -rf plyvel", cwd=base_dir)
call("git clone https://github.com/wbolster/plyvel", cwd=base_dir)
plyvel_dir = base_dir.joinpath('plyvel')
call("git checkout master", cwd=plyvel_dir)
call("make", cwd=plyvel_dir)
call("python setup.py install", cwd=plyvel_dir)
shutil.rmtree(plyvel_dir)
