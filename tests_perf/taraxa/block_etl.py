from taraxa.ethereum_etl import export_blocks
from taraxa.type_util import *


class BlockEtl:

    def __init__(self, on_block: Callable[[dict], NoReturn], ethereum_etl_opts=fdict()):
        self._on_block = on_block
        self.ethereum_etl_opts = ethereum_etl_opts
        self._finished_blocks = {}
        self._next_block = 0

    def run(self, from_block, to_block, page_size):
        self._next_block = from_block
        print(f'running block etl with args {self.ethereum_etl_opts}')
        print(f'block_count: {self._next_block}')
        while self._next_block <= to_block:
            self._download(self._next_block, min(self._next_block + page_size, to_block))

    def _store_block(self, block):
        block_num = block['number']
        if block_num != self._next_block:
            self._finished_blocks[block_num] = block
            # print('downloaded future block')
            return
        block_to_flush = block
        while block_to_flush is not None:
            self._on_block(block_to_flush)
            self._next_block += 1
            print(f'block_count: {self._next_block}')
            block_to_flush = self._finished_blocks.pop(self._next_block, None)

    def _download(self, from_block, to_block):
        print(f'downloading blocks {from_block}-{to_block}...')
        export_blocks(from_block, to_block, self._store_block, **self.ethereum_etl_opts)
