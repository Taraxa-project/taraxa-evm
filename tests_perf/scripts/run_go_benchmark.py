import json
from os import cpu_count
from os import environ
from pathlib import Path

from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LOCAL_LIB_PACKAGE
from taraxa.util import call

data_dir = Path('/mnt/xvdf')
blockchain_data_dir = data_dir.joinpath('perf_test')
project_name = Path(__file__).stem
project_dir = data_dir.joinpath(project_name)
project_dir.mkdir(parents=True, exist_ok=True)
profile_dir = project_dir.joinpath('profiles')
profile_dir.mkdir(parents=True, exist_ok=True)
benchmark_config_path = project_dir.joinpath("benchmark_config.json")

block_db_conf = rocksdb_util.Config(read_only=True, path=str(blockchain_data_dir.joinpath('blocks')))
block_db = BlockDB(block_db_conf.new_db())

min_tx_to_process = 5000
from_block = 4735000
state_transition_request = {
    'stateRoot': block_db.get_block(from_block - 1)['stateRoot'],
}
state_transition_block = state_transition_request['block'] = map_block(block_db.get_block(from_block))
state_transition_transactions = state_transition_block['transactions']
for _, db_block in block_db.iteritems(from_block + 1):
    if len(state_transition_transactions) >= min_tx_to_process:
        break
    block = map_block(db_block)
    state_transition_transactions.extend(block['transactions'])
    state_transition_block['gasLimit'] += block['gasLimit']

benchmark_config_path.write_text(json.dumps({
    'vmConfig': {
        'readDB': {
            'cacheSize': 8192,
            'db': {
                'type': 'rocksdb',
                'options': {
                    'file': str(blockchain_data_dir.joinpath('ethereum_emulated_state_rocksdb')),
                    'readOnly': True
                }
            }
        },
        'writeDB': {
            'cacheSize': 8192,
            'db': {
                'type': 'rocksdb',
                'options': {
                    'file': str(project_dir.joinpath('dummy_state')),
                }
            }
        },
        'blockDB': {
            'type': 'rocksdb',
            'options': {
                'file': str(block_db_conf.path),
                'readOnly': True
            }
        },
        'conflictDetectorInboxPerTransaction': 500,
        'threadPool': {
            'threadCount': 0,
            'queueSize': 0
        }
    },
    'stateTransitionRequest': state_transition_request,
}))

call(f"go test "
     "-timeout 99999m "
     "-benchtime 5s "
     "-count 3 "
     f"-cpuprofile {profile_dir.joinpath('cpu.prof')} "
     f"-memprofile {profile_dir.joinpath('mem.prof')} "
     f"-mutexprofile {profile_dir.joinpath('mutex.prof')} "
     f"-blockprofile {profile_dir.joinpath('block.prof')} "
     f"-bench StateTransitionTestMode",
     cwd=str(LOCAL_LIB_PACKAGE.joinpath('taraxa_vm')),
     env={
         **environ,
         'CONFIG_FILE': str(benchmark_config_path),
         'GOGC': 'off'
     })
