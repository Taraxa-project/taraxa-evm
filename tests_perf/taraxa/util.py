import subprocess
import json


def call(cmd, *args, **kwargs):
    assert 0 == subprocess.call(cmd.split(' '), *args, **kwargs)


def read_json(path, **kwargs):
    if not path.exists():
        return
    with open(path) as f:
        return json.load(f, **kwargs)


def write_json(path, value, **kwargs):
    with open(path, mode='w') as f:
        json.dump(value, f, **kwargs)
