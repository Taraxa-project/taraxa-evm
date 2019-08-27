import rocksdb

from apps.blockchain_data import BlockDB

block_db = BlockDB(
    rocksdb.DB('/workspace/data/ethereum_blockchain_mainnet_rocksdb',
               rocksdb.Options(create_if_missing=True),
               read_only=True))

for k, v in block_db.iteritems():
    print(f"{k} : {v}")
