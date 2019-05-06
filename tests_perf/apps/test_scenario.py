from pathlib import Path

from apps import shell
from apps.blockchain_data import collect_blockchain_data
from apps.compute_state_db import execute_transactions

MAX_BLOCK = 7665710


def subdir(base, child):
    return str(Path(base).resolve().joinpath(child))


@shell.command
def collect_blocks(base_path: str):
    Path(base_path).mkdir(parents=True, exist_ok=True)
    collect_blockchain_data(subdir(base_path, 'blocks'), subdir(base_path, 'block_hashes'),
                            to_block=MAX_BLOCK,
                            page_size=500000,
                            ethereum_etl_opts=dict(
                                provider_uri='https://mainnet.infura.io/v3/6f560dcc0ff74bc0aa6596b7fe253573',
                                # provider_uri='https://ethereum.api.nodesmith.io/v1/mainnet/jsonrpc?apiKey=783e2979af6d4e6a8e4475d31830948d',
                                batch_size=750,
                                parallelism_factor=3,
                                timeout=500))


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

# collect_blocks('../out')
