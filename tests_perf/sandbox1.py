from os.path import expanduser

import rocksdb

from apps.blockchain_data import BlockDB

block_db = BlockDB(rocksdb.DB(f'{expanduser("~")}/data',
                              rocksdb.Options(create_if_missing=True),
                              read_only=True))

for k, v in block_db.iteritems():
    print(f'{k}: {v}')
