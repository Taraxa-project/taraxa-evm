from . import shell
from taraxa.typing import fdict
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from tempfile import tempdir
from pathlib import Path
from taraxa.leveldb import LevelDB
import rocksdb


@shell.command
def compute_state_db(library_dir=None, block_db_path=None, block_hash_db_path=None):
    library_dir = library_dir or Path(tempdir()).joinpath('.taraxa_vm')
    library_file = Path(library_dir).joinpath('taraxa_vm.so')
    LibTaraxaEvm.build(library_file)
    lib_taraxa_vm = LibTaraxaEvm(library_file)
    block_hash_ldb = LevelDB(Path(block_hash_db_path).resolve(), create_if_missing=True)
    block_ldb = ()
    block_hash_ldb, block_ldb = (LevelDB(p, create_if_missing=True) for p in (block_hash_db_path, block_db_path))
    pass
