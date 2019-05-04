from pathlib import Path

from . import shell
from .blockchain_data import collect_blockchain_data
from .compute_state_db import execute_transactions

MAX_BLOCK = 7665710


def subdir(base, child):
    return str(Path(base).resolve().joinpath(child))


@shell.command
def collect_blocks(base_path: str):
    collect_blockchain_data(subdir(base_path, 'blocks'), subdir(base_path, 'block_hashes'), to_block=MAX_BLOCK)


@shell.command
def compute_state_eth_mode(base_path: str):
    execute_transactions(
        vm_opts={
            'stateDB': {
                'cacheSize': 2048,
                'ldb': {
                    'file': subdir(base_path, 'ethereum_emulated_state'),
                    'cache': 1024,
                    'handles': 1024,
                },
            },
            'blockHashLDB': {
                'file': subdir(base_path, 'block_hashes'),
                'cache': 256,
                'handles': 256,
            },
        },
        to_block=MAX_BLOCK,
        emulate_ethereum=True,
        block_db_opts={
            'path': subdir(base_path, 'blocks')
        },
        target_result_db_opts={
            'path': subdir(base_path, 'ethereum_emulated_results')
        })
