import plyvel

from contextlib import contextmanager
from pathlib import Path


class SessionRequiredError(Exception):
    pass


class LevelDB:

    def __init__(self, path, *openargs, **openkwargs):
        self.path = Path(path)
        self.openargs = openargs
        self.openkwargs = openkwargs
        self._db = None

    @contextmanager
    def open_session(self):
        try:
            self._db = plyvel.DB(str(self.path), *self.openargs, **self.openkwargs)
            yield self.session
        finally:
            try:
                self._db.close()
            finally:
                self._db = None

    @property
    def session(self):
        if self._db is None:
            raise SessionRequiredError(f"Session is not opened for this leveldb at {self.path}")
        return self._db
