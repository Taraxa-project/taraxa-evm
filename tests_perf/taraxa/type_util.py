import typing
from typing import *

from frozendict import frozendict


class _FDictProxy(frozendict):

    def __repr__(self):
        return str(self._dict)


def fdict(*args, **kwargs) -> Mapping:
    return _FDictProxy(*args, **kwargs)


opts = fdict


class NamedTuple(typing.NamedTuple):

    @classmethod
    def defaults(cls) -> Mapping:
        return opts(cls._field_defaults)
