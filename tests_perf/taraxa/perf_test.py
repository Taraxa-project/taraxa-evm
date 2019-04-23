import os

from .block_db import BlockDatabase
from .block_hash_db import BlockHashDatabase
from paths import *
from .taraxa_evm import TaraxaEvm
from . import util

out_dir = base_dir.joinpath('out')
taraxa_c_lib_path = out_dir.joinpath('taraxa_evm_c_lib').joinpath('taraxa_evm.so')
ethereum_etl_dir = out_dir.joinpath('ethereum_etl')
state_db_path, blockchain_db_path = (out_dir.joinpath(e) for e in ('state_db', 'blockchain_db'))
block_db_dir = out_dir.joinpath('block_db')
progress_file = out_dir.joinpath('progress.json')


def run(*args, until_block=10000000, **kwargs):
    for path in (state_db_path, blockchain_db_path):
        os.makedirs(path, exist_ok=True)
    # if not taraxa_c_lib_path.exists():
    #     TaraxaEvm.build_c_lib(taraxa_c_lib_path)
    print("Compiling VM C library...")
    TaraxaEvm.build_c_lib(taraxa_c_lib_path)
    taraxa_evm = TaraxaEvm(taraxa_c_lib_path)
    block_hash_db = BlockHashDatabase(str(blockchain_db_path))
    block_db = BlockDatabase(block_db_dir, page_size=5000, download_batch_size=5000)
    state_root, next_block = util.read_json(progress_file) or [
        "0x0000000000000000000000000000000000000000000000000000000000000000",
        # first block with transactions
        46147
    ]
    while next_block < until_block:
        print(f"processing block: {next_block}")
        block, transactions = block_db.get_block_and_tx(next_block)
        block_hash_db.put_block_hash(next_block, block['hash'])
        if transactions:
            sequential_set = list(range(len(transactions)))
            request = {
                "stateTransition": {
                    "stateRoot": state_root,
                    "block": _map_block(block),
                    "transactions": [_map_transaction(tx) for tx in transactions],
                },
                "stateDatabase": {
                    "file": str(state_db_path),
                    "cache": 0,
                    "handles": 0
                },
                "blockchainDatabase": {
                    "file": str(blockchain_db_path),
                    "cache": 0,
                    "handles": 0
                },
                "concurrentSchedule": {
                    "sequential": sequential_set
                }
            }
            print(f"tx set: {transactions}")
            print(f'VM request : {request}')
            result = taraxa_evm.run(request)
            print(f'VM result: {result}')
            err = result.get('error')
            if err:
                raise RuntimeError(err)
            state_root = result['stateTransitionResult']['stateRoot']
        next_block = next_block + 1
        util.write_json(progress_file, [state_root, next_block])


def _map_block(block):
    return {
        "coinbase": block['miner'],
        "number": str(block['number']),
        "time": str(block['timestamp']),
        "difficulty": str(block['difficulty']),
        "gasLimit": int(block['gas_limit']),
        "hash": block['hash']
    }


def _map_transaction(transaction):
    return {
        "to": transaction['to_address'],
        "from": transaction['from_address'],
        "nonce": transaction['nonce'],
        "amount": str(transaction['value']),
        "gasLimit": transaction['gas'],
        "gasPrice": str(transaction['gas_price']),
        "data": transaction['input'],
        "hash": transaction['hash']
    }
