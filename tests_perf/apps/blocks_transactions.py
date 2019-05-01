from multiprocessing import cpu_count

import rocksdb

from taraxa.block_etl import BlockEtl, BlockDB
from taraxa.typing import fdict
from . import shell


@shell.command
def collect_blocks_and_tx(db_path: str,
                          to_block=7665710, page_size=500000, rocksdb_opts=fdict(),
                          ethereum_etl_opts=fdict(
                              batch_size=500,
                              max_workers=cpu_count() * 3,
                              timeout=15)):
    db = rocksdb.DB(db_path, rocksdb.Options(create_if_missing=True, **rocksdb_opts))
    BlockEtl(BlockDB(db), ethereum_etl_opts=ethereum_etl_opts).run(to_block, page_size)


@shell.command
def validate_blocks_and_tx_db(db_path: str, rocksdb_opts=fdict()):
    db = rocksdb.DB(db_path, rocksdb.Options(**rocksdb_opts), read_only=True)
    BlockDB(db).validate()
