from frozendict import frozendict
from typing import *


class _fdict_proxy(frozendict):

    def __repr__(self):
        return str(self._dict)


def fdict(*args, **kwargs) -> Mapping:
    return _fdict_proxy(*args, **kwargs)
