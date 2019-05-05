import json
from pathlib import Path
from tempfile import gettempdir

import rocksdb

from taraxa.rocksdb_util import ceil_entry
from .blockchain_data import BlockDB
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.type_util import *
from taraxa.util import raise_if_not_none
from taraxa.context_util import current_exit_stack, with_exit_stack
from . import shell
from taraxa import rocksdb_util


class TaraxaVMResultDB:
    Key = BlockDB.Key
    Value = BlockDB.Value

    def __init__(self, db: rocksdb.DB):
        self._db = db

    def put(self, block_num: Key, result: Value):
        self._db.put(BlockDB.block_key_encode(block_num), json.dumps(result).encode())

    def get(self, block_num: Key) -> Value:
        value = self._db.get(BlockDB.block_key_encode(block_num))
        if value:
            return json.loads(value)

    def ceil_entry(self) -> Tuple[Key, Value]:
        entry = ceil_entry(self._db)
        if entry:
            return BlockDB.block_key_decode(entry[0]), json.loads(entry[1])


@shell.command
@with_exit_stack
def execute_transactions(vm_opts,
                         from_block=0,
                         to_block=None,
                         emulate_ethereum=False,
                         vm_lib_dir=Path(gettempdir()).joinpath('.taraxa_vm'),
                         block_db_opts=rocksdb_util.Config.defaults(),
                         target_result_db_opts=rocksdb_util.Config.defaults(),
                         source_result_db_opts=rocksdb_util.Config.defaults()):
    exit_stack = current_exit_stack()
    library_file = Path(vm_lib_dir).joinpath('taraxa_vm.so')
    print(f'Building the vm library as: {library_file}')
    LibTaraxaEvm.build(library_file)
    lib_taraxa_vm = LibTaraxaEvm(library_file)
    taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", vm_opts)
    raise_if_not_none(err, RuntimeError)
    taraxa_vm_ptr = lib_taraxa_vm.as_ptr(taraxa_vm_handle)
    exit_stack.enter_context(taraxa_vm_ptr.scope())
    block_db = BlockDB(rocksdb_util.Config(read_only=True, **block_db_opts).new_db())
    source_result_db = (source_result_db_opts and TaraxaVMResultDB(rocksdb_util.Config(
        read_only=True,
        **source_result_db_opts
    ).new_db()))
    target_result_db = TaraxaVMResultDB(rocksdb_util.Config(**{
        'opts': {
            'create_if_missing': True
        },
        **target_result_db_opts
    }).new_db())
    block_num, last_result = target_result_db.ceil_entry() or (from_block - 1, None)
    block_num += 1
    for block_num_from_db, block in block_db.iteritems(from_block=block_num):
        if to_block and block_num > to_block:
            break
        assert block_num_from_db == block_num
        base_result = source_result_db.get(block_num - 1) if source_result_db else last_result
        state_root = base_result['state_transition_result']['stateRoot'] if base_result else '0x' + '0' * 64
        transactions = block['transactions']
        tx_count = len(transactions)
        print(f'Processing block {block_num}, tx count: {tx_count}, base_state_root: {state_root}')
        state_transition = {
            "stateRoot": state_root,
            "block": _map_block(block),
            "transactions": [_map_transaction(tx) for tx in transactions],
        }
        if emulate_ethereum:
            concurrent_schedule = {
                "sequential": list(range(tx_count))
            }
        else:
            concurrent_schedule, err = taraxa_vm_ptr.call('GenerateSchedule', state_transition)
            raise_if_not_none(err, lambda e: RuntimeError(f'Schedule generation failed: {e}'))
        state_transition_result, err = taraxa_vm_ptr.call("TransitionState", state_transition, concurrent_schedule)
        raise_if_not_none(err, lambda e: RuntimeError(f'State transition failed: {e}'))
        last_result = {
            'concurrent_schedule': concurrent_schedule,
            'state_transition_result': state_transition_result,
        }
        target_result_db.put(block_num, last_result)
        block_num += 1
    assert not to_block or block_num - 1 == to_block


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
