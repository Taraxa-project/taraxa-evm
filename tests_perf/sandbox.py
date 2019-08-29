import rocksdb

from apps.blockchain_data import BlockDB

from_block = 4821013

block_db = BlockDB(
    rocksdb.DB('/workspace/data/eth_mainnet_rocksdb/blockchain',
               rocksdb.Options(create_if_missing=True),
               read_only=True))

block_db_to = BlockDB(
    rocksdb.DB(f'/workspace/data/eth_mainnet_rocksdb/blockchain_{from_block}_{from_block + 1000000}',
               rocksdb.Options(create_if_missing=True)))

# state_db = rocksdb.DB(f'/workspace/data/ethereum_state_mainnet_rocksdb',
#                       rocksdb.Options(create_if_missing=True),
#                       read_only=True)

for k, v in block_db.iteritems(from_block=from_block):
    block_db_to.put_block(v)
    # state_root = v['stateRoot']
    # state_root_bytes = bytes.fromhex(state_root.replace('0x', ''))
    # if not state_db.get(state_root_bytes):
    #     break
    # print(state_root)
    print(k)
