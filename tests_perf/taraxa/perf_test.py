import json
import os

from paths import *
from taraxa import util
from taraxa.block_db import BlockDatabase
from taraxa.leveldb import LevelDB
from taraxa.lib_taraxa_evm import LibTaraxaEvm

out_dir = base_dir.joinpath('out')
taraxa_c_lib_path = out_dir.joinpath('taraxa_evm_c_lib').joinpath('taraxa_evm.so')
ethereum_state_db_path, taraxa_state_db_path, block_hash_db_path, block_db_path, progress_file_path = \
    (out_dir.joinpath(e)
     for e in ('state_db', 'taraxa_state_db', 'block_hash_db', 'block_db', 'progress.json'))


def run(*args,
        # last block in 2017
        until_block=4832685,
        **kwargs):
    for path in (ethereum_state_db_path, block_hash_db_path):
        os.makedirs(path, exist_ok=True)
    # if not taraxa_c_lib_path.exists():
    #     TaraxaEvm.build_c_lib(taraxa_c_lib_path)
    print("Compiling VM C library...")
    LibTaraxaEvm.build(taraxa_c_lib_path)
    lib_taraxa_evm = LibTaraxaEvm(taraxa_c_lib_path)
    block_hash_ldb, block_ldb = (LevelDB(p, create_if_missing=True)
                                 for p in (block_hash_db_path, block_db_path))
    block_db = BlockDatabase(block_ldb, page_size=20000, download_batch_size=5000)
    taraxa_state_db_config = {
        "file": str(taraxa_state_db_path),
        "cache": 0,
        "handles": 0
    }
    taraxa_evm_ptr, block_hash_db_ptr, err = lib_taraxa_evm.call("NewVM", {
        'stateDB': {
            'leveldb': {
                "file": str(ethereum_state_db_path),
                "cache": 256,
                "handles": 256
            },
            'cacheSize': 4096 * 64
        },
        'externalApi': {
            'blockHashLevelDB': {
                "file": str(block_hash_db_path),
                "cache": 256,
                "handles": 256
            }
        }
    })
    if err is not None:
        raise RuntimeError(err)
    progress_file_path.touch()
    with util.ContextManagers(
            block_db.open_session(),
            lib_taraxa_evm.as_ptr(taraxa_evm_ptr).scope(),
            lib_taraxa_evm.as_ptr(block_hash_db_ptr).scope()) as (_, taraxa_evm, block_hash_db), \
            progress_file_path.open(mode='r+') as progress_file:
        state_root, next_block = json.loads(progress_file.read() or "[]") or [
            "0x0000000000000000000000000000000000000000000000000000000000000000",
            # first block with transactions
            46147
        ]
        while next_block <= until_block:
            print(f"processing block: {next_block}")
            block, transactions = block_db.get_block_and_tx(next_block)
            block_hash_db.call('Put', next_block, block['hash'])
            print(f"tx count: {len(transactions)}")
            if transactions:
                state_transition = {
                    "stateRoot": state_root,
                    "block": _map_block(block),
                    "transactions": [_map_transaction(tx) for tx in transactions],
                }
                # print("Generating concurrent schedule...")
                # schedule_response = taraxa_evm.run({
                #     "stateTransition": state_transition,
                #     "stateDatabase": ethereum_state_db_config,
                #     "blockHashDatabase": block_hash_db_config,
                # })
                # print("Running state transition using the schedule...")
                # taraxa_state_transition_response = taraxa_evm.run({
                #     "stateTransition": state_transition,
                #     "stateDatabase": ethereum_state_db_config,
                #     "blockHashDatabase": block_hash_db_config,
                #     "stateTransitionTargetDatabase": taraxa_state_db_config,
                #     "concurrentSchedule": schedule_response["concurrentSchedule"]
                # })
                # print("Running state transition in Ethereum mode...")
                eth_state_transition_result, err = taraxa_evm.call("TransitionState", {
                    "stateTransition": state_transition,
                    "concurrentSchedule": {
                        "sequential": list(range(len(transactions)))
                    },
                })
                if err is not None:
                    raise RuntimeError(err)
                state_root = eth_state_transition_result['stateRoot']
            else:
                pass
                # print("No transactions - skipping")
            next_block = next_block + 1
            progress_file.seek(0)
            progress_file.write(json.dumps([state_root, next_block]))


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
