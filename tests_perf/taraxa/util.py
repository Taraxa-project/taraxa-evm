import json
import subprocess
from typing import Type


def popen(cmd, *args, **kwargs):
    return subprocess.Popen(cmd.split(' '), *args, **kwargs)


def call(cmd, *args, **kwargs):
    assert 0 == subprocess.call(cmd.split(' '), *args, **kwargs)


def raise_if_not_none(err, factory: Type[BaseException] = None):
    if err is not None:
        if factory is None:
            raise err
        raise factory(err)


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
    return str(obj).encode()
