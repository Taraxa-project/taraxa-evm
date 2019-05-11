import json
from pathlib import Path
from tempfile import gettempdir

import rocksdb

from taraxa.rocksdb_util import ceil_entry
from .blockchain_data import BlockDB
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.type_util import *
from taraxa.util import raise_if_not_none, assert_eq
from taraxa.context_util import current_exit_stack, with_exit_stack
from . import shell
from taraxa import rocksdb_util
from ethereumetl.utils import hex_to_dec


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
        state_root = base_result['stateTransitionResult']['stateRoot'] if base_result else '0x' + '0' * 64
        transactions = block['transactions']
        tx_count = len(transactions)
        print(f'Processing block {block_num}, tx count: {tx_count}, base_state_root: {state_root}')
        state_transition = {
            "stateRoot": state_root,
            "block": map_block(block),
            "expectedRoot": block['stateRoot']
        }
        if emulate_ethereum:
            concurrent_schedule = {
                "sequential": list(range(tx_count))
            }
        else:
            concurrent_schedule, err = taraxa_vm_ptr.call('GenerateSchedule', state_transition)
            raise_if_not_none(err, lambda e: RuntimeError(f'Schedule generation failed: {e}'))
        state_transition_result, metrics, err = taraxa_vm_ptr.call("TransitionStateLikeEthereum",
                                                                   state_transition, concurrent_schedule)
        raise_if_not_none(err, lambda e: RuntimeError(f'State transition failed: {e}'))
        print(f'metrics: {json.dumps(map_metrics(metrics))}'.ljust(20, '='))
        # print(json.dumps(metrics))
        for i, receipt in enumerate(state_transition_result['receipts'] or []):
            eth_receipt = receipt['ethereumReceipt']
            expected_receipt = block['transactions'][i]['receipt']
            if 'status' in expected_receipt:
                assert_eq(eth_receipt['status'], expected_receipt['status'])
            if 'root' in expected_receipt:
                assert_eq(eth_receipt.get['root'], expected_receipt.get['root'])
            assert_eq(eth_receipt['gasUsed'], expected_receipt['gasUsed'])
            assert_eq(eth_receipt['cumulativeGasUsed'], expected_receipt['cumulativeGasUsed'])
            assert_eq(eth_receipt['contractAddress'],
                      expected_receipt['contractAddress'] or '0x0000000000000000000000000000000000000000')
        assert_eq(state_transition_result['usedGas'], hex_to_dec(block['gasUsed']))
        assert_eq(state_transition_result['stateRoot'], block['stateRoot'])

        last_result = {
            'concurrentSchedule': concurrent_schedule,
            'stateTransitionResult': state_transition_result,
        }
        target_result_db.put(block_num, last_result)
        block_num += 1
    assert not to_block or block_num - 1 == to_block


def map_block(block):
    return {
        "number": block['number'],
        "coinbase": block['miner'],
        "time": hex_to_dec(block['timestamp']),
        "difficulty": hex_to_dec(block['difficulty']),
        "gasLimit": hex_to_dec(block['gasLimit']),
        "hash": block['hash'],
        "transactions": [map_transaction(tx) for tx in block['transactions']],
        "uncles": [map_uncle(u) for u in block['uncleBlocks']]
    }


def map_uncle(uncle_block):
    return {
        'number': hex_to_dec(uncle_block['number']),
        'coinbase': uncle_block['miner']
    }


def map_transaction(transaction):
    return {
        "to": transaction['to'],
        "from": transaction['from'],
        "nonce": hex_to_dec(transaction['nonce']),
        "amount": hex_to_dec(transaction['value']),
        "gasLimit": hex_to_dec(transaction['gas']),
        "gasPrice": hex_to_dec(transaction['gasPrice']),
        "data": transaction['input'],
        "hash": transaction['hash']
    }


def map_metrics(metrics: Mapping):
    tx_totals = {}
    tx_metrics = metrics['transactionMetrics']
    for tx_record in tx_metrics:
        tx_record_enriched = {
            **tx_record,
            'pct_trie_reads': pct(tx_record['trieReads'], tx_record['totalExecutionTime']),
            'pct_persistent_reads_in_trie_reads': pct(tx_record['persistentReads'], tx_record['trieReads']),
        }
        for k, v in tx_record_enriched.items():
            tx_totals[k] = tx_totals.get(k, 0) + v
    return {
        'pct_tx_execution': pct(tx_totals['totalExecutionTime'], metrics['totalTime']),
        'pct_persistent_commit': pct(metrics['persistentCommit'], metrics['totalTime']),
        'pct_pct_trie_commit_sync': pct(metrics['trieCommitSync'], metrics['totalTime']),
        'pct_pct_trie_commit_total': pct(metrics['trieCommitTotal'], metrics['totalTime']),
        'tx_averages': {k: round(v / len(tx_metrics), 5) for k, v in tx_totals.items()}
    }


def pct(x, y):
    return round(x / y if y != 0 else 0.000000, 5)
