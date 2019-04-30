import json
import subprocess
from contextlib import AbstractContextManager
from typing import ContextManager


def popen(cmd, *args, **kwargs):
    return subprocess.Popen(cmd.split(' '), *args, **kwargs)


def call(cmd, *args, **kwargs):
    assert 0 == subprocess.call(cmd.split(' '), *args, **kwargs)


def read_str(path):
    if not path.exists():
        return
    with open(path) as f:
        return f.read()


def write_str(path, value):
    with open(path, mode='w') as f:
        f.write(value)


def read_json(path, *args, **kwargs):
    val = read_str(path)
    return None if val is None else json.loads(val, *args, **kwargs)


def write_json(path, value, *args, **kwargs):
    write_str(path, json.dumps(value, *args, **kwargs))


def str_bytes(obj):
    return str(obj).encode(encoding='utf-8')


class ContextManagers(AbstractContextManager):

    def __init__(self, *delegates: ContextManager):
        self.delegates = delegates

    def __enter__(self):
        return [ctx.__enter__() for ctx in self.delegates]

    def __exit__(self, exc_type, exc_value, traceback):
        for ctx in self.delegates:
            ctx.__exit__(exc_type, exc_value, traceback)
