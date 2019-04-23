import json
import os

from .ethereum_etl import export_blocks_and_transactions
from paths import *


class BlockDatabase:

    def __init__(self, state_dir, page_size=20000, download_batch_size=5000):
        self.state_dir = Path(state_dir)
        os.makedirs(self.state_dir, exist_ok=True)
        self.download_size = page_size
        self.batch_size = download_batch_size
        self.blocks_file_path = self.state_dir.joinpath('blocks.json')
        self.transactions_file_path = self.state_dir.joinpath('transactions.json')
        self.block_cache = {}
        self.tx_cache = {}

    def get_block_and_tx(self, number):
        if not self.blocks_file_path.exists():
            self._download_batch(number)
        elif not self.block_cache:
            self._reload_cache()
        block = self.block_cache.get(number)
        if not block:
            self._download_batch(number)
        block = self.block_cache.get(number)
        if block:
            return block, self.tx_cache.get(number, [])

    def _reload_cache(self):
        self.block_cache = {}
        with open(self.blocks_file_path) as f:
            for line in f:
                block = json.loads(line)
                self.block_cache[block['number']] = block
        self.tx_cache = {}
        with open(self.transactions_file_path) as f:
            for line in f:
                tx = json.loads(line)
                self.tx_cache.setdefault(tx['block_number'], []).append(tx)

    def _download_batch(self, from_block):
        to_block = from_block + self.download_size - 1
        print(f"downloading blocks {from_block}-{to_block}...")
        export_blocks_and_transactions(from_block, to_block, self.blocks_file_path, self.transactions_file_path,
                                       batch_size=self.batch_size)
        self._reload_cache()
