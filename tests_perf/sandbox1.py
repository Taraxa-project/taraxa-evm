import rocksdb

from apps.blockchain_data import BlockDB

block_db = BlockDB(rocksdb.DB('/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain',
                              rocksdb.Options(
                                  create_if_missing=True,
                                  max_open_files=-1
                              ),
                              read_only=True))

for k, v in block_db.iteritems(from_block=0):
    print(k)
