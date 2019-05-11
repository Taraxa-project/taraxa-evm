import json
from pathlib import Path
from shutil import rmtree
from tempfile import gettempdir

from ethereumetl.executors.bounded_executor import BoundedExecutor

from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.util import raise_if_not_none

intervals = [
    (778483, 934646), (1620940, 1801799), (2912407, 3100154), (3800776, 3955159)
]

# BASE_DIR = Path('out')
BASE_DIR = Path('/mnt/xvdf/perf_test/')
BASE_DIR.mkdir(exist_ok=True, parents=True)

dummy_state_dir = BASE_DIR.joinpath(f'ethereum_emulated_state_metrics_dummies')
dummy_state_dir.mkdir(exist_ok=True, parents=True)

block_db_conf = rocksdb_util.Config(read_only=True, path=str(BASE_DIR.joinpath('blocks')))
block_db = BlockDB(block_db_conf.new_db())

print("building lib...")
library_path = Path(gettempdir()).joinpath(f'taraxa_vm_parallel').joinpath('taraxa_vm.so')
rmtree(library_path, ignore_errors=True)
LibTaraxaEvm.build(library_path)
lib_taraxa_vm = LibTaraxaEvm(library_path)


def new_vm(partition):
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
            'type': 'memory',
            'options': {
                # 'file': str(dummy_state_dir.joinpath(f'state_db_{partition}')),
            }
        },
        'blockDB': {
            'type': 'rocksdb',
            'options': {
                'file': block_db_conf.path,
                'readOnly': True
            }
        },
    }
    print('building vm...')
    taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", conf)
    raise_if_not_none(err, RuntimeError)
    print(f'build vm # {partition}')
    return lib_taraxa_vm.as_ptr(taraxa_vm_handle)

    # contracts_count = 0
    # contract_totals = {
    #     'total': 0,
    #     'subtotals': {
    #         'trieReads': 0,
    #         'persistentReads': 0,
    #     }
    # }
    # blocks_count = 0
    # block_totals = {
    #     'total': 0,
    #     'subtotals': {
    #         'trieCommit': 0,
    #         'persistentCommit': 0,
    #     },
    # }


partitions = 6
vm_instances = {}
executors = [BoundedExecutor(100, max_workers=1) for i in range(partitions)]


def process_block(block_num):
    try:
        partition = block_num % partitions
        taraxa_vm_ptr = vm_instances.get(partition)
        if not taraxa_vm_ptr:
            taraxa_vm_ptr = new_vm(partition)
            vm_instances[partition] = taraxa_vm_ptr
        prev_block = block_db.get_block(block_num - 1)
        block = block_db.get_block(block_num)
        state_transition_result, metrics, err = taraxa_vm_ptr.call("TransitionStateLikeEthereum",
                                                                   {
                                                                       "stateRoot": prev_block['stateRoot'],
                                                                       "block": map_block(block),
                                                                       "expectedRoot": block['stateRoot']
                                                                   },
                                                                   {
                                                                       "sequential": list(
                                                                           range(len(block['transactions'])))
                                                                   })
        raise_if_not_none(err, lambda e: RuntimeError(f'State transition failed: {e}'))
        return {
            'blockNumber': block_num,
            'metrics': metrics,
            **state_transition_result,
        }
    except BaseException as e:
        print(f'error: {e}')
        raise e


results_db = rocksdb_util.Config(path=str(BASE_DIR.joinpath('eth_metrics_parallel')), opts={
    'create_if_missing': True
}).new_db()
last_block_bytes, _ = rocksdb_util.ceil_entry(results_db) or (None, None)
next_block = BlockDB.block_key_decode(last_block_bytes) + 1 if last_block_bytes else intervals[0][0]

total = sum(end - start + 1 for start, end in intervals)
processed = 0

result_cache = {}


def on_result(result):
    global next_block, result_cache, processed
    result_cache[result['blockNumber']] = result
    while True:
        block = result_cache.pop(next_block, None)
        if block is None:
            break
        results_db.put(BlockDB.block_key_encode(next_block), json.dumps(block).encode())
        print(f'processed block {next_block}, progress: {processed / total}%')
        processed += 1
        next_block += 1


for start, end in intervals:
    block_num = start - 1
    while True:
        block_num += 1
        if block_num < next_block:
            processed += 1
            print(f'replaying done work, progress: {processed / total}%')
            continue
        if block_num > end:
            break
        executor = executors[block_num % partitions]
        executor.submit(process_block, block_num).add_done_callback(lambda f: on_result(f.result()))

for e in executors:
    e.shutdown(wait=True)
