import json

import rocksdb

from taraxa.block_etl import BlockEtl
from taraxa.context_util import with_exit_stack, current_exit_stack
from taraxa.leveldb import LevelDB
from taraxa.rocksdb_util import ceil_entry
from taraxa.type_util import Iterable, Tuple, Dict
from taraxa.type_util import fdict
from . import shell


def validate_block(block):
    block_num = block['number']
    for i, tx in enumerate(block['transactions']):
        assert tx['hash'] == tx['receipt']['transactionHash']
        assert tx['blockNumber'] == block_num
        assert tx['transactionIndex'] == i
    for i, h in enumerate(block['uncles']):
        assert h == block['uncleBlocks'][i]['hash']


class BlockDB:
    Key = int
    Value = Dict

    def __init__(self, db: rocksdb.DB):
        self._db = db

    @staticmethod
    def block_key_encode(block_num: Key) -> bytes:
        return str(block_num).zfill(9).encode()

    @staticmethod
    def block_key_decode(key: bytes) -> Key:
        return int(str(key, encoding='utf-8').lstrip('0') or '0')

    def put_block(self, block: Value):
        block_num = block['number']
        self._db.put(self.block_key_encode(block_num), json.dumps(block).encode())

    def max_block_num(self) -> Key:
        key, _ = ceil_entry(self._db) or (None, None)
        if key:
            return self.block_key_decode(key)

    def get_block(self, key: Key):
        enc = self._db.get(self.block_key_encode(key))
        return enc and json.loads(enc)

    def iteritems(self, from_block: Key = None) -> Iterable[Tuple[Key, Value]]:
        itr = self._db.iteritems()
        if from_block:
            itr.seek(self.block_key_encode(from_block))
        else:
            itr.seek_to_first()
        return ((self.block_key_decode(k), json.loads(v)) for k, v in itr)


@shell.command
@with_exit_stack
def collect_blockchain_data(block_db_path: str, block_hash_db_path: str,
                            to_block=7665710, page_size=1000000,
                            ethereum_etl_opts=fdict(
                                batch_size=500,
                                parallelism_factor=2.5,
                                timeout=20)):
    block_db = BlockDB(rocksdb.DB(block_db_path, rocksdb.Options(create_if_missing=True)))
    block_hash_db = LevelDB(block_hash_db_path, create_if_missing=True)
    current_exit_stack().enter_context(block_hash_db.open_session())

    def on_block(block):
        validate_block(block)
        block_db.put_block(block)
        block_hash_db.session.put(str(block['number']).encode(), block['hash'].encode())

    last_block = block_db.max_block_num()
    from_block = last_block + 1 if last_block else 0
    BlockEtl(on_block, ethereum_etl_opts=ethereum_etl_opts).run(from_block, to_block, page_size)


@shell.command
# @with_exit_stack
def ensure_blockchain_data_valid(db_path: str, from_block=0):
    block_db = BlockDB(rocksdb.DB(db_path, rocksdb.Options(), read_only=True))
    # block_hash_db = LevelDB(block_hash_db_path, create_if_missing=True)
    # block_hash_db.repair()
    # current_exit_stack().enter_context(block_hash_db.open_session())
    expected_block_num = from_block
    for block_num_db, block in block_db.iteritems(from_block):
        print(f'validating block: {expected_block_num}')
        block_num = block['number']
        assert block_num_db == block_num == expected_block_num
        validate_block(block)
        # block_hash_db_key = str(block_num).encode()
        # block_hash_bytes = block['hash'].encode()
        # block_hash_from_db = block_hash_db.session.get(block_hash_db_key)
        # if block_hash_from_db != block_hash_bytes:
        #     if not block_hash_from_db:
        #         print('detected absent entry in the block hash db, fixing')
        #     else:
        #         print('wrong block hash in the db, fixing')
        #     block_hash_db.session.put(block_hash_db_key, block_hash_bytes)
        expected_block_num += 1
