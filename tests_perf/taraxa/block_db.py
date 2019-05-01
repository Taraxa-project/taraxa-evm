import json

import rocksdb

from taraxa.typing import Iterable, Tuple, Dict

Block = Dict


class BlockDB:

    def __init__(self, db: rocksdb.DB):
        self._db = db

    @staticmethod
    def block_key_encode(block_num: int):
        return str(block_num).zfill(9).encode()

    @staticmethod
    def block_key_decode(key: bytes) -> int:
        return int(str(key, encoding='utf-8').lstrip('0') or '0')

    def put_block(self, block: Block):
        self._db.put(self.block_key_encode(block['number']), json.dumps(block).encode())

    def max_block_num(self) -> int:
        itr = self._db.iterkeys()
        itr.seek_to_last()
        for k in itr:
            return self.block_key_decode(k)

    def iteritems(self, from_block: int = None) -> Iterable[Tuple[int, Block]]:
        itr = self._db.iteritems()
        itr.seek_to_first()
        if from_block:
            itr.seek(self.block_key_encode(from_block))
        return ((self.block_key_decode(k), json.loads(v)) for k, v in itr)

    def validate(self):
        expected_block_num = 0
        for block_num_db, block in self.iteritems():
            print(f'validating block: {expected_block_num}')
            block_num = block['number']
            assert block_num_db == block_num == expected_block_num
            tx_count = block['transaction_count']
            transactions = block['transactions']
            assert tx_count == len(transactions)
            for i in range(tx_count):
                tx = transactions[i]
                assert tx['block_number'] == block_num
                assert tx['transaction_index'] == i
            expected_block_num += 1
