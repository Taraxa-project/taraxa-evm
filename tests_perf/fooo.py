from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from pathlib import Path
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.util import raise_if_not_none
from tempfile import gettempdir

block_db_path = '/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain'
library_file = Path(gettempdir()).joinpath('taraxa_vm.so')
LibTaraxaEvm.build(library_file)
lib_taraxa_vm = LibTaraxaEvm(library_file)
taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", {
    'stateDB': {
        'cacheSize': 1024,
        'db': {
            'type': 'rocksdb',
            'options': {
                'file': '/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/state',
                'readOnly': True,
                'maxOpenFiles': 1000,
            }
        },
    },
    'writeDB': {
        'type': 'memory'
    },
    'blockDB': {
        'type': 'rocksdb',
        'options': {
            'file': block_db_path,
            'readOnly': True,
            'maxOpenFiles': 1000,
        }
    },
})
raise_if_not_none(err, RuntimeError)
taraxa_vm_ptr = lib_taraxa_vm.as_ptr(taraxa_vm_handle)
block_db = BlockDB(rocksdb_util.Config(read_only=True, path=block_db_path, opts={'max_open_files': 1000}).new_db())
next_block = 1000128
prev_block = block_db.get_block(next_block - 1)
while next_block <= 2000000:
    print(f'Processing block {next_block}')
    block = block_db.get_block(next_block)
    req = {
        "stateRoot": prev_block["stateRoot"],
        "block": map_block(block),
        "expectedRoot": block['stateRoot'],
    }
    print(req)
    state_transition_result, _, err = taraxa_vm_ptr.call("TransitionStateLikeEthereum", req, {})
    raise_if_not_none(err, lambda e: RuntimeError(f'State transition failed: {e}'))
    print(state_transition_result["stateRoot"], block['stateRoot'])
    assert state_transition_result["stateRoot"] == block['stateRoot']
    next_block += 1
    prev_block = block
    break
