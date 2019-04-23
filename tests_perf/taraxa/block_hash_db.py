import plyvel


class BlockHashDatabase:

    def __init__(self, leveldb_path):
        self.leveldb_path = leveldb_path

    def put_block_hash(self, block_number, block_hash_hex_str):
        ldb = plyvel.DB(self.leveldb_path, create_if_missing=True)
        try:
            ldb.put(str(block_number).encode(), block_hash_hex_str.encode())
        finally:
            ldb.close()
