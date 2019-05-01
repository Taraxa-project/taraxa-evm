from taraxa.ethereum_etl import export_blocks_and_transactions
from taraxa.typing import fdict
from .block_db import BlockDB


class BlockEtl:

    def __init__(self, db: BlockDB, ethereum_etl_opts=fdict()):
        self._db = db
        self.ethereum_etl_opts = ethereum_etl_opts
        self._block_cache = {}
        self._tx_cache = {}
        self._finished_blocks = {}
        self._next_block = 0

    def run(self, to_block, page_size):
        self._next_block = self._db.max_block_num() + 1
        print(f'block_count: {self._next_block}')
        while self._next_block <= to_block:
            self._download(self._next_block, min(self._next_block + page_size, to_block))

    def _try_flush(self, block_num):
        block = self._block_cache.get(block_num)
        if not block:
            return
        transactions = self._tx_cache.get(block_num, {})
        block_tx_count = block['transaction_count']
        if block_tx_count != len(transactions):
            return
        self._block_cache.pop(block_num)
        self._tx_cache.pop(block_num, None)
        self._finished_blocks[block_num] = {
            **block,
            'transactions': [transactions[i] for i in range(block_tx_count)]
        }
        while True:
            block_to_write = self._finished_blocks.pop(self._next_block, None)
            if not block_to_write:
                break
            self._db.put_block(block_to_write)
            self._next_block += 1
            print(f'block_count: {self._next_block}')

    def _store_block(self, block):
        block_num = block['number']
        self._block_cache[block_num] = block
        self._try_flush(block_num)

    def _store_tx(self, tx):
        block_num = tx['block_number']
        self._tx_cache.setdefault(block_num, {})[tx['transaction_index']] = tx
        self._try_flush(block_num)

    def _download(self, from_block, to_block):
        print(f'downloading blocks {from_block}-{to_block}...')
        export_blocks_and_transactions(from_block, to_block,
                                       on_block=self._store_block, on_transaction=self._store_tx,
                                       **self.ethereum_etl_opts)
