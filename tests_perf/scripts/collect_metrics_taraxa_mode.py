import json
from pathlib import Path
from tempfile import gettempdir
from zipfile import ZipFile, ZIP_DEFLATED

from ethereumetl.utils import hex_to_dec

from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.util import raise_if_not_none


def map_metrics(metrics):
    tx_totals = {}
    tx_metrics = metrics['transactionMetrics']
    for tx_record in tx_metrics:
        tx_record_enriched = {
            **tx_record,
            'pct_trie_reads': pct(tx_record['trieReads'], tx_record['totalExecutionTime']),
            'pct_persistent_reads_in_trie_reads': pct(tx_record['persistentReads'], tx_record['trieReads']),
        }
        for k, val in tx_record_enriched.items():
            tx_totals[k] = tx_totals.get(k, 0) + val
    return {
        'pct_tx_execution': pct(tx_totals.get('totalExecutionTime', 0), metrics['totalTime']),
        'pct_persistent_commit': pct(metrics['persistentCommit'], metrics['totalTime']),
        'pct_pct_trie_commit_sync': pct(metrics['trieCommitSync'], metrics['totalTime']),
        'pct_pct_trie_commit_total': pct(metrics['trieCommitTotal'], metrics['totalTime']),
        'trie_commit_speedup': ((metrics['totalTime'] - metrics['trieCommitSync'] + metrics['trieCommitTotal'])
                                / metrics['totalTime']),
        'tx_averages': {k: round(val / len(tx_metrics), 5) for k, val in tx_totals.items()}
    }


def pct(x, y):
    return round(x / y if y != 0 else 0.000000, 5)


intervals = [
    # (778483, 808909),
    # (1620940, 1657167),
    # (2912407, 2948852),
    # (3800776, 3831962),
    # (4652926, 4728185)
    # (4795000, 4797000)
    (4795000, 4800000)
]

# BASE_DIR = Path('out')
BASE_DIR = Path('/mnt/xvdf/perf_test/')

PROJECT_NAME = 'taraxa_metrics_4'

BASE_DIR.mkdir(exist_ok=True, parents=True)

dummy_state_dir = BASE_DIR.joinpath(f'{PROJECT_NAME}_dummy_state')
dummy_state_dir.mkdir(exist_ok=True, parents=True)

block_db_conf = rocksdb_util.Config(read_only=True, path=str(BASE_DIR.joinpath('blocks')))
block_db = BlockDB(block_db_conf.new_db())


def new_vm(partition):
    print("building lib")
    library_path = Path(gettempdir()).joinpath(f'{PROJECT_NAME}_taraxa_vm_{partition}').joinpath('taraxa_vm.so')
    LibTaraxaEvm.build(library_path)
    lib_taraxa_vm = LibTaraxaEvm(library_path)
    print("built lib")

    conf = {
        'stateDB': {
            'cacheSize': 0,
            'db': {
                'type': 'rocksdb',
                'options': {
                    'file': str(BASE_DIR.joinpath('ethereum_emulated_state_rocksdb')),
                    'readOnly': True
                }
            }
        },
        'writeDB': {
            'type': 'rocksdb',
            'options': {
                'file': str(dummy_state_dir.joinpath(f'state_db_{partition}')),
            }
        },
        'blockDB': {
            'type': 'rocksdb',
            'options': {
                'file': block_db_conf.path,
                'readOnly': True
            }
        },
        'conflictDetectorInboxPerTransaction': 500
    }
    taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", conf)
    raise_if_not_none(err, RuntimeError)
    return lib_taraxa_vm.as_ptr(taraxa_vm_handle)


partitions = 1
vm_instances = {}


def process_block(block_num):
    partition = block_num % partitions
    taraxa_vm_ptr = vm_instances.get(partition)
    if not taraxa_vm_ptr:
        taraxa_vm_ptr = new_vm(partition)
        vm_instances[partition] = taraxa_vm_ptr
    prev_block = block_db.get_block(block_num - 1)
    block = block_db.get_block(block_num)
    state_transition_request = {
        "stateRoot": prev_block['stateRoot'],
        "block": map_block(block),
        "expectedRoot": block['stateRoot']
    }
    transactions = block['transactions']
    tx_count = len(transactions)
    # print(json.dumps(transactions))
    schedule, schedule_metrics, err = taraxa_vm_ptr.call("GenerateSchedule", state_transition_request)
    raise_if_not_none(err)
    schedule['sequential'] = schedule.get('sequential') or []
    # seq_tx_count = len(schedule['sequential'])
    # print(f'tx_count: {tx_count}, parallel tx %: {pct(tx_count - seq_tx_count, tx_count)}')
    # print(json.dumps(metrics))
    state_transition_result, state_transition_metrics, err = taraxa_vm_ptr.call("TransitionState",
                                                                                state_transition_request,
                                                                                schedule)
    # {"sequential": list(range(len(transactions)))})
    raise_if_not_none(err)
    eth_result, total_time_eth, err = taraxa_vm_ptr.call("RunLikeEthereum",
                                                         state_transition_request)
    raise_if_not_none(err)
    print(eth_result['stateRoot'], block['stateRoot'])
    assert eth_result['stateRoot'] == block['stateRoot']
    print(total_time_eth)
    # print(json.dumps(map_metrics(metrics)))
    # raise RuntimeError("foo")
    # contract_transactions = []
    # for tx_index, receipt in enumerate(state_transition_result.get('receipts') or []):
    #     gas_used = hex_to_dec(receipt['ethereumReceipt']['gasUsed'])
    #     if gas_used != 0:
    #         contract_transactions.append(tx_index)
    #     else:
    #         print("non-contract tx")
    r = {
        'blockNumber': block_num,
        'txCount': tx_count,
        'schedule': schedule,
        # 'contractTransactions': contract_transactions,
        'metrics': {
            'totalTimeEth': total_time_eth,
            'scheduleGeneration': schedule_metrics,
            'stateTransition': state_transition_metrics,
        },
    }
    return r


results_db = rocksdb_util.Config(path=str(BASE_DIR.joinpath(PROJECT_NAME)), opts={
    'create_if_missing': True
}).new_db()
last_block_bytes, _ = rocksdb_util.ceil_entry(results_db) or (None, None)
total = sum(end - start + 1 for start, end in intervals)
processed = 0
for start, end in intervals:
    current_block = start
    while current_block <= end:
        if not results_db.get(BlockDB.block_key_encode(current_block)):
            print(f'processing block {current_block}, progress: {processed / total}%')
            result = process_block(current_block)
            results_db.put(BlockDB.block_key_encode(current_block), json.dumps(result).encode())
        current_block += 1
        processed += 1

itr = results_db.itervalues()
itr.seek_to_first()

import statistics

speedups = []
speedups_norm = []
parallel_tx_ratios = []
parallel_tx_speedups = []
tx_execution_ratios = []
trie_commit_sync_ratios = []
persistent_commit_ratios = []
post_actions_rations = []
trie_commit_speedups = []
min_tx_count = 0
max_tx_count = 1000
for v in itr:
    result = json.loads(v)
    tx_count = result['txCount']
    if not (min_tx_count <= tx_count <= max_tx_count):
        continue
    sequential_tx = set(result['schedule']['sequential'])
    state_transition_metrics = result['metrics']['stateTransition']
    tx_metrics = state_transition_metrics['transactionMetrics']
    parallel_tx_total_time = sum(tx_metrics[i]['totalExecutionTime'] for i in range(tx_count) if i not in sequential_tx)
    total_time = state_transition_metrics['totalTime']
    trie_commit_sync = state_transition_metrics['trieCommitSync']
    trie_commit_total = state_transition_metrics['trieCommitTotal']
    total_time_norm = total_time - trie_commit_sync + trie_commit_total
    parallel_tx_sync_time = state_transition_metrics['parallelTransactionsSync']
    time_if_parallel_were_sequential = total_time - parallel_tx_sync_time + parallel_tx_total_time
    speedups.append(time_if_parallel_were_sequential / total_time)
    speedups_norm.append(result['metrics']['totalTimeEth'] / total_time)
    parallel_tx_speedups.append(parallel_tx_total_time / parallel_tx_sync_time)
    parallel_tx_ratios.append((tx_count - len(sequential_tx)) / tx_count)
    sequential_tx_time = state_transition_metrics['sequentialTransactions']
    tx_execution_ratios.append((sequential_tx_time + parallel_tx_sync_time) / total_time)
    trie_commit_sync_ratios.append(trie_commit_sync / total_time)
    persistent_commit_ratios.append(state_transition_metrics['persistentCommit'] / total_time)
    post_actions_rations.append((state_transition_metrics['postProcessingSync'] +
                                 state_transition_metrics['conflictDetectionSync']) /
                                total_time)
    trie_commit_speedups.append(trie_commit_total / trie_commit_sync)

print(
    f'block count: {len(speedups)}\n'
    f'min speedup_norm: {min(speedups_norm)}\n'
    f'max speedup_norm: {max(speedups_norm)}\n'
    f'mean speedup_norm: {statistics.mean(speedups_norm)}\n'
    f'median speedup_norm: {statistics.median(speedups_norm)}\n'
    f'min speedup: {min(speedups)}\n'
    f'max speedup: {max(speedups)}\n'
    f'mean speedup: {statistics.mean(speedups)}\n'
    f'median speedup: {statistics.median(speedups)}\n'
    f'mean % parallel tx: {statistics.mean(parallel_tx_ratios)}\n'
    f'mean parallel tx speedup: {statistics.mean(parallel_tx_speedups)}\n'
    f'mean % tx execution: {statistics.mean(tx_execution_ratios)}\n'
    f'mean % trie commit: {statistics.mean(trie_commit_sync_ratios)}\n'
    f'mean % persistent commit: {statistics.mean(persistent_commit_ratios)}\n'
    f'mean % postprocessing + conflict detection: {statistics.mean(post_actions_rations)}\n'
    f'mean trie commit speedup: {statistics.mean(trie_commit_speedups)}\n'
    f'min trie commit speedup: {min(trie_commit_speedups)}\n'
    f'max trie commit speedup: {max(trie_commit_speedups)}\n'
)

# archive_path = BASE_DIR.joinpath(f'{PROJECT_NAME}.zip')
# archive_path.touch(exist_ok=True)
# archive_file = ZipFile(archive_path, mode='w', compression=ZIP_DEFLATED, allowZip64=True)
# results_file = archive_file.open('results.json', mode='w', force_zip64=True)
# i = 0
# for v in itr:
#     results_file.write(v + b'\n')
#     print(f'zipping progress: {i / total}%')
#     # print(str(v, encoding='utf-8'))
#     i += 1
# results_file.close()
# archive_file.close()
