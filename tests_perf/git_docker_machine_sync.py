#!usr/bin/env python3
import sys
import tempfile
from fnmatch import fnmatch
from io import TextIOWrapper
from pathlib import Path
from subprocess import Popen, PIPE, call
from typing import Iterable


def any_matches(path, patterns):
    for pattern in patterns:
        if fnmatch(path, pattern):
            return True


def run(machine, dest, *exclude_patterns):
    from zipfile import ZipFile, ZIP_DEFLATED

    dest = Path(dest)
    repo_dir = None
    with Popen('git rev-parse --show-toplevel'.split(' '), stdout=PIPE) as p:
        base_dir_str = TextIOWrapper(p.stdout, encoding="utf-8").read().rstrip()
        repo_dir = Path(base_dir_str)
        assert p.wait() == 0
    archive_name = f'{repo_dir.name}.zip'
    archive_path_local = Path(tempfile.gettempdir()).joinpath(archive_name).resolve()
    archive_file = ZipFile(archive_path_local, mode='w', compression=ZIP_DEFLATED, compresslevel=9)
    with archive_file, Popen('git ls-files'.split(" "), stdout=PIPE, cwd=repo_dir) as p:
        files: Iterable[str] = TextIOWrapper(p.stdout, encoding="utf-8")
        for rel_path in files:
            rel_path = rel_path.rstrip()
            if any_matches(rel_path, exclude_patterns):
                continue
            src = repo_dir.joinpath(rel_path)
            if not src.exists() or src.is_dir():
                continue
            archive_file.write(src, arcname=rel_path)
        assert p.wait() == 0
        archive_path_dest = dest.joinpath(archive_name)
    target_dir = dest.joinpath(repo_dir.name)
    assert 0 == call(f'docker-machine scp -d {archive_path_local} {machine}:{archive_path_dest}'.split(" "))
    assert 0 == call(f'docker-machine ssh {machine} sudo '
                     # f'rm -rf {target_dir} && '
                     f'unzip -o -d {target_dir} {archive_path_dest}'
                     .split(' '))


run(*sys.argv[1:])

# docker-machine create --driver amazonec2 --amazonec2-open-port 8000 --amazonec2-region us-east-2 --amazonec2-vpc-id vpc-0c890d64 --amazonec2-instance-type c4.8xlarge --amazonec2-access-key AKIAX4HKAES7EFKQ5H7Z --amazonec2-secret-key 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 vm-tests-2
