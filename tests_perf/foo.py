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
        self._tx_left = {}
        self._current_block = 0

    @staticmethod
    def _block_key(block_num):
        return pad_block_num(block_num).encode()

    def _prepare(self):
        itr = self._db.iteritems()
        itr.seek_to_first()
        current_block_stored = self._db.get(self.CURRENT_BLOCK_KEY)
        if current_block_stored:
            self._current_block = int(current_block_stored)
            itr.seek(self._block_key(self._current_block - 1))
        print(f'current_block: {self._current_block}')
        for k, v in itr:
            if k == self.CURRENT_BLOCK_KEY:
                continue
            split = k.split(b'_')
            block_num = int(split[0])
            tx_delta = -1
            is_block = len(split) == 1
            if is_block:
                tx_delta = json.loads(v)['transaction_count']
            block_finished = self._add_transactions_left(block_num, tx_delta)
            if block_finished:
                assert self._current_block != block_num
            if block_finished and self._current_block < block_num:
                self._download(self._current_block, block_num - 1)

    def _add_transactions_left(self, block_number, cnt):
        tx_left = self._tx_left.get(block_number, 0) + cnt
        if tx_left != 0:
            self._tx_left[block_number] = tx_left
            return
        while self._current_block == block_number or self._tx_left.get(self._current_block) == 0:
            self._tx_left.pop(self._current_block, None)
            self._current_block += 1
            self._db.put(self.CURRENT_BLOCK_KEY, str(self._current_block).encode())
            print(f'current_block: {self._current_block}')
        return True

    def _store_block(self, block):
        block_num = block['number']
        self._db.put(self._block_key(block_num), json.dumps(block).encode())
        self._add_transactions_left(block_num, block['transaction_count'])

    def _store_tx(self, tx):
        block_num = tx['block_number']
        self._db.put(f"{pad_block_num(block_num)}_{pad_tx_index(tx['transaction_index'])}".encode(),
                     json.dumps(tx).encode())
        self._add_transactions_left(block_num, -1)

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
    batch_size=5 * 1000,
    max_workers=cpu_count() * 4,
    timeout=15 * 1000
))
 .run())
