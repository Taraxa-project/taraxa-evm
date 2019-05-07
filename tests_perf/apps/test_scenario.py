from pathlib import Path

from apps import shell
from apps.blockchain_data import collect_blockchain_data, ensure_blockchain_data_valid
from apps.compute_state_db import execute_transactions
from taraxa.type_util import *

MAX_BLOCK = 7665710


def subdir(base, child):
    return str(Path(base).resolve().joinpath(child))


@shell.command
def collect_blocks(base_path: str):
    Path(base_path).mkdir(parents=True, exist_ok=True)
    collect_blockchain_data(subdir(base_path, 'blocks'), subdir(base_path, 'block_hashes'),
                            to_block=MAX_BLOCK,
                            page_size=600000,
                            ethereum_etl_opts=dict(
                                provider_uri='https://mainnet.infura.io/v3/6f560dcc0ff74bc0aa6596b7fe253573',
                                # provider_uri='https://ethereum.api.nodesmith.io/v1/mainnet/jsonrpc?apiKey=783e2979af6d4e6a8e4475d31830948d',
                                batch_size=200,
                                parallelism_factor=3,
                                timeout=500))


def compute_eth_state(base_path, state_db_config_factory, suffix=''):
    block_db_path = subdir(base_path, 'blocks')
    state_db_path = subdir(base_path, f'ethereum_emulated_state{suffix}')
    execute_transactions(
        vm_opts={
            'stateDB': {
                'cacheSize': 2048,
                'db': state_db_config_factory(state_db_path),
            },
            'blockDB': {
                'type': 'rocksdb',
                'options': {
                    'file': block_db_path,
                    'readOnly': True
                }
            },
        },
        to_block=MAX_BLOCK,
        emulate_ethereum=True,
        block_db_opts={
            'path': block_db_path
        },
        target_result_db_opts={
            'path': subdir(base_path, f'ethereum_emulated_results{suffix}')
        })


@shell.command
def compute_state_eth_mode_leveldb(base_path: str):
    compute_eth_state(base_path,
                      lambda path: {
                          'type': 'leveldb',
                          'options': {
                              'file': path,
                              'cache': 1024,
                              'handles': 1024,
                          }
                      })


@shell.command
def compute_state_eth_mode_rocksdb(base_path: str):
    compute_eth_state(base_path,
                      lambda path: {
                          'type': 'rocksdb',
                          'options': {
                              'file': path,
                          }
                      },
                      suffix='_rocksdb')


@shell.command
def ensure_block_data_valid(base_path: str, from_block=0):
    ensure_blockchain_data_valid(subdir(base_path, 'blocks'), from_block)

# collect_blocks('../out')
