import json
from pathlib import Path
from tempfile import gettempdir

from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.util import raise_if_not_none
import rocksdb
from zipfile import ZipFile, ZIP_DEFLATED

intervals = [
    (778483, 808909), (1620940, 1657167), (2912407, 2948852), (3800776, 3831962)
]

# BASE_DIR = Path('out')
BASE_DIR = Path('/mnt/xvdf/perf_test/')

PROJECT_NAME = 'eth_metrics'

BASE_DIR.mkdir(exist_ok=True, parents=True)

dummy_state_dir = BASE_DIR.joinpath(f'ethereum_emulated_state_metrics_dummies')
dummy_state_dir.mkdir(exist_ok=True, parents=True)

block_db_conf = rocksdb_util.Config(read_only=True, path=str(BASE_DIR.joinpath('blocks')))
block_db = BlockDB(block_db_conf.new_db())


def new_vm(partition):
    print("building lib")
    library_path = Path(gettempdir()).joinpath(f'taraxa_vm_{partition}').joinpath('taraxa_vm.so')
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
    }
    taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", conf)
    raise_if_not_none(err, RuntimeError)
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


partitions = 1
vm_instances = {}


# executors = [ProcessPoolExecutor(max_workers=1) for i in range(partitions)]


def process_block(block_num):
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


results_db = rocksdb_util.Config(path=str(BASE_DIR.joinpath(PROJECT_NAME)), opts={
    'create_if_missing': True
}).new_db()
last_block_bytes, _ = rocksdb_util.ceil_entry(results_db) or (None, None)
last_block = BlockDB.block_key_decode(last_block_bytes) if last_block_bytes else -1
total = sum(end - start + 1 for start, end in intervals)
processed = 0
for start, end in intervals:
    current_block = start
    while True:
        if current_block > end:
            break
        if current_block > last_block:
            print(f'processing block {current_block}, progress: {processed / total}%')
            result = process_block(current_block)
            results_db.put(BlockDB.block_key_encode(current_block), json.dumps(result).encode())
        current_block += 1
        processed += 1

itr = results_db.itervalues()
itr.seek_to_first()
archive_path = BASE_DIR.joinpath(f'{PROJECT_NAME}.zip')
archive_path.touch(exist_ok=True)

archive_file = ZipFile(archive_path, mode='w', compression=ZIP_DEFLATED, allowZip64=True)
results_file = archive_file.open('results.json', mode='w', force_zip64=True)
i = 0
for v in itr:
    results_file.write(v + b'\n')
    print(f'zipping progress: {i / total}%')
    # print(str(v, encoding='utf-8'))
    i += 1
results_file.close()
archive_file.close()
