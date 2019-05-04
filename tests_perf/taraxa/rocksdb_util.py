import rocksdb

from taraxa.type_util import *


class Config(typing.NamedTuple):
    path: str = ''
    opts: dict = {}
    column_families: dict = None
    read_only: bool = False

    def new_db(self):
        return rocksdb.DB(self.path, rocksdb.Options(**self.opts), self.column_families, self.read_only)

    @staticmethod
    def defaults():
        return fdict()


def ceil_entry(db: rocksdb.DB) -> Tuple[bytes, bytes]:
    itr = db.iteritems()
    itr.seek_to_last()
    for k, v in itr:
        return k, v
