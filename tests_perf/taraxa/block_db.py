import json

from taraxa.ethereum_etl import export_blocks_and_transactions
from taraxa.leveldb import LevelDB
from taraxa.util import str_bytes


class CorruptedBlockException(Exception):
    pass


class BlockDatabase:

    def __init__(self, ldb: LevelDB, page_size=20000, download_batch_size=5000):
        self.page_size = page_size
        self.download_batch_size = download_batch_size
        self._ldb = ldb
        self._reset_cache()

    def open_session(self):
        return self._ldb.open_session()

    def get_block_and_tx(self, number):
        if not self._block_cache:
            self._load_cache(number)
        block = self._block_cache.get(number)
        if not block:
            self._download_batch(number)
        block = self._block_cache.get(number)
        if block:
            return block, self._tx_cache.get(number, [])

    def _reset_cache(self):
        self._block_cache = {}
        self._tx_cache = {}

    def _load_cache(self, from_block):
        self._reset_cache()
        block_number = from_block
        while True:
            block_bytes = self._ldb.session.get(str_bytes(block_number))
            if not block_bytes:
                break
            block = json.loads(block_bytes)
            self._cache_block(block)
            for tx_index in range(block['transaction_count']):
                tx_bytes = self._ldb.session.get(self._tx_ldb_key(block_number, tx_index))
                if not tx_bytes:
                    self._unload_block(block_number)
                    raise CorruptedBlockException()
                self._cache_tx(json.loads(tx_bytes))
            block_number = block_number + 1

    def _cache_block(self, block):
        self._block_cache[block['number']] = block

    def _cache_tx(self, tx):
        self._tx_cache.setdefault(tx['block_number'], []).append(tx)

    def _unload_block(self, number):
        self._ldb.session.delete(str_bytes(number))
        self._block_cache.pop(number, None)
        self._tx_cache.pop(number, None)

    @staticmethod
    def _tx_ldb_key(block_num, index):
        return f"{block_num}_{index}".encode()

    def _download_batch(self, from_block):
        to_block = from_block + self.page_size - 1
        print(f"downloading blocks {from_block}-{to_block}...")
        self._reset_cache()

        def save_block(block):
            number = block['number']
            print(number)
            self._ldb.session.put(str_bytes(number), json.dumps(block).encode())
            self._cache_block(block)

        def save_transaction(tx):
            key = self._tx_ldb_key(tx['block_number'], tx['transaction_index'])
            self._ldb.session.put(key, json.dumps(tx).encode())
            self._cache_tx(tx)

        export_blocks_and_transactions(from_block, to_block,
                                       batch_size=self.download_batch_size,
                                       on_block=save_block,
                                       on_transaction=save_transaction)
