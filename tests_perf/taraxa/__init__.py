import json
from ctypes import *
from pathlib import Path
from subprocess import call

this_dir = Path(__file__).parent
go_lib_src = this_dir.parent.parent.joinpath('main')

# build the library
call(f"go build -tags=lib_cpp -buildmode=c-shared -o {this_dir.joinpath('taraxa_evm.so')}".split(" "), cwd=go_lib_src)

lib = cdll.LoadLibrary(this_dir.joinpath('taraxa_evm.so'))
lib.RunEvm.argtypes = [c_char_p]
lib.RunEvm.restype = c_char_p

json_str = json.dumps({
    "stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "block": {
        "coinbase": "0x0000000000000000000000000000000000000063",
        "number": "0",
        "time": "0",
        "difficulty": "0",
        "gasLimit": 100000000000,
        "hash": "0x0000000000000000000000000000000000000000000000000000000000000000"
    },
    "transactions": [
        {
            "to": None,
            "from": "0x0000000000000000000000000000000000000064",
            "nonce": 0,
            "amount": "0",
            "gasLimit": 100000000,
            "gasPrice": "0",
            "data": "0x608060405234801561001057600080fd5b5060d58061001f6000396000f3fe60806040523480"
                    "15600f57600080fd5b506004361060325760003560e01c80631ab06ee51460375780639507d39"
                    "a146059575b600080fd5b605760048036036040811015604b57600080fd5b5080359060200135"
                    "6085565b005b607360048036036020811015606d57600080fd5b50356097565b6040805191825"
                    "2519081900360200190f35b60009182526020829052604090912055565b6000908152602081905"
                    "260409020549056fea165627a7a72305820c82be8219f3d83f7714c7c102df669c9a7ee75af"
                    "d5a816633add76013ac211ec0029",
            "hash": "0x0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "to": None,
            "from": "0x0000000000000000000000000000000000000065",
            "nonce": 0,
            "amount": "0",
            "gasLimit": 100000000,
            "gasPrice": "0",
            "data": "0x608060405234801561001057600080fd5b5060d58061001f6000396000f3fe6080604052348"
                    "015600f57600080fd5b506004361060325760003560e01c80631ab06ee51460375780639507d39"
                    "a146059575b600080fd5b605760048036036040811015604b57600080fd5b50803590602001356"
                    "085565b005b607360048036036020811015606d57600080fd5b50356097565b604080519182525"
                    "19081900360200190f35b60009182526020829052604090912055565b600090815260208190526"
                    "0409020549056fea165627a7a72305820c82be8219f3d83f7714c7c102df669c9a7ee75afd5a816"
                    "633add76013ac211ec0029",
            "hash": "0x0000000000000000000000000000000000000000000000000000000000000000"
        }
    ],
    "ldbConfig": {
        "file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/main/state_transition/__test_ldb__",
        "cache": 0, "handles": 0
    },
    "concurrentSchedule": None
})

# call evm
ret = lib.RunEvm(json_str.encode(encoding='utf-8'))
print(str(ret, encoding='utf-8'))
