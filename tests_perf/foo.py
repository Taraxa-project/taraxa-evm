import json
from multiprocessing import cpu_count

import rocksdb

from taraxa.ethereum_etl import export_blocks_and_transactions


def pad_block_num(number):
    return str(number).zfill(9)


def pad_tx_index(number):
    return str(number).zfill(6)


def padded_to_int(number_str):
    return int(number_str.lstrip('0') or '0')


class BlockAndTxExporter:
    CURRENT_BLOCK_KEY = b'current_block'

    def __init__(self, db: rocksdb.DB, to_block, **opts):
        self.to_block = to_block
        self.opts = {**dict(
            ethereum_etl=dict(
                batch_size=4 * 1000,
                max_workers=cpu_count() * 2,
                timeout=15 * 1000
            )
        ), **opts}
        self._db = db
        self._block_cache = {}
        self._tx_cache = {}
        self._tx_left = {}
        self._current_block = 0
        self._block_count = 0

    @staticmethod
    def _block_key(block_num):
        return pad_block_num(block_num).encode()

    def _prepare(self):
        itr = self._db.iteritems()
        itr.seek_to_first()
        current_block_stored = self._db.get(self.CURRENT_BLOCK_KEY)
        if current_block_stored:
            self._current_block = int(current_block_stored)
            self._block_count = self._current_block
            itr.seek(self._block_key(self._current_block - 1))
        print(f'block_count: {self._block_count}')
        for k, v in itr:
            if k == self.CURRENT_BLOCK_KEY:
                continue
            split = k.split(b'_')
            block_num = int(split[0])
            tx_delta = -1
            is_block = len(split) == 1
            if is_block:
                tx_delta = json.loads(v)['transaction_count']
            block_finished = self._sync(block_num, tx_delta)
            if block_finished:
                assert self._current_block != block_num
            if block_finished and self._current_block < block_num:
                self._download(self._current_block, block_num - 1)

    def _sync(self, block_num):
        block = self._block_cache[block_num]
        transactions = self._tx_cache.setdefault(block_num, {})
        block_tx_count = block['transaction_count']
        if block_tx_count != len(transactions):
            return
        self._block_cache.pop(block_num)
        self._tx_cache.pop(block_num)
        block['transactions'] = [transactions[i] for i in range(block_tx_count)]
        self._db.put(self._block_key(block_num), json.dumps(block).encode())
        if self._current_block == block_num:
            self._current_block += 1
            self._db.put(self.CURRENT_BLOCK_KEY, str(self._current_block).encode())
            self._sync(self._current_block)
            print(f'current_block: {self._current_block}')
        print(f'block_count: {self._current_block}')
        return True

    def _store_block(self, block):
        block_num = block['number']
        self._block_cache[block_num] = block
        self._sync(block_num)

    def _store_tx(self, tx):
        block_num = tx['block_number']
        self._tx_cache.setdefault(block_num, {})[tx['transaction_index']] = tx
        self._sync(block_num)

    def _download(self, from_block, to_block):
        print(f'exporting blocks {from_block}-{to_block}...')
        export_blocks_and_transactions(from_block, to_block,
                                       on_block=self._store_block, on_transaction=self._store_tx,
                                       **self.opts['ethereum_etl'])

    def run(self):
        self._prepare()
        if self._current_block < self.to_block:
            self._download(self._current_block, self.to_block)


db = rocksdb.DB('/mnt/xvdf/blocks_tx', rocksdb.Options(create_if_missing=True))
(BlockAndTxExporter(db, 7665710, ethereum_etl=dict(
    batch_size=3 * 1000,
    max_workers=cpu_count() * 3,
    timeout=15 * 1000
))
 .run())
