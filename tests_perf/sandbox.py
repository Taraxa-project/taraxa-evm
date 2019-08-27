import rocksdb

from apps.blockchain_data import BlockDB

from_block = 2176170

block_db = BlockDB(
    rocksdb.DB('/workspace/data/ethereum_blockchain_mainnet_rocksdb',
               rocksdb.Options(create_if_missing=True),
               read_only=True))

block_db_to = BlockDB(
    rocksdb.DB(f'/workspace/data/ethereum_blockchain_mainnet_rocksdb_{from_block}_{block_db.max_block_num()}',
               rocksdb.Options(create_if_missing=True)))

for k, v in block_db.iteritems(from_block=from_block):
    block_db_to.put_block(v)
