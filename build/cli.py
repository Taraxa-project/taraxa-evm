import os
import platform
import shutil
import sys
from os import path
from os.path import join, abspath
from subprocess import call

PROJECT_NAME = 'go-ethereum'

THIS_DIR = path.dirname(path.abspath(__file__))
ROOT_DIR = join(THIS_DIR, path.pardir)
DEV_DEPS_DIR = join(THIS_DIR, 'dependencies')
PIP_DEPS_DIR = join(DEV_DEPS_DIR, 'pip')
PROTOC_DIR = join(DEV_DEPS_DIR, 'protoc')
PROTOC_EXECUTABLE = join(PROTOC_DIR, 'bin', 'protoc')
BIN_DIR = join(THIS_DIR, 'bin')
MAIN_PACKAGE = join(ROOT_DIR, 'main')
API_PACKAGE = join(MAIN_PACKAGE, 'api')
FINAL_EXECUTABLE = join(BIN_DIR, 'evm')

PROTOC_VERSION = '3.6.1'

if not path.isdir(DEV_DEPS_DIR):
    call(['pip', 'install', f'--target={PIP_DEPS_DIR}', '-r', 'requirements.txt'], cwd=THIS_DIR)
sys.path.append(THIS_DIR)
sys.path.append(PIP_DEPS_DIR)
ENTRY_POINT = __import__('click').group(name='build_cli')(lambda: None)


def command(*args, **kwargs):
    def register(fn):
        ENTRY_POINT.command(*args, **kwargs)(fn)
        return fn

    return register


def _call(*args, **kwargs):
    assert 0 == call(args, **kwargs)


@command()
def clean_tmp():
    from glob import iglob
    for line in open(join(ROOT_DIR, '.tmp_files')).readlines():
        path_pattern = line.split('# ')[0].strip()
        if path_pattern.startswith('/'):
            path_pattern = path_pattern[1:]
        if not path_pattern or path_pattern == '#':
            continue
        for file_path in iglob(abspath(join(ROOT_DIR, path_pattern)), recursive=True):
            print(f'Removing {file_path}')
            if path.isdir(file_path):
                shutil.rmtree(file_path)
            else:
                os.remove(file_path)


@command()
def install_protoc():
    from io import BytesIO
    from zipfile import ZipFile
    from urllib.request import urlopen
    import stat
    os_map = {
        'Darwin': 'osx'
    }
    arch_map = {
        'x86_64': 'x86_64'
    }
    protoc_os = os_map.get(platform.system())
    protoc_arch = arch_map.get(platform.machine())
    url = (f'https://github.com/protocolbuffers/protobuf/releases/download'
           f'/v{PROTOC_VERSION}/protoc-{PROTOC_VERSION}-{protoc_os}-{protoc_arch}.zip')
    print(f'Downloading protoc from {url}')
    r = urlopen(url)
    ZipFile(BytesIO(r.read())).extractall(PROTOC_DIR)
    os.chmod(PROTOC_EXECUTABLE, os.stat(PROTOC_EXECUTABLE).st_mode | stat.S_IEXEC)


@command()
def grpc_compile():
    _call(PROTOC_EXECUTABLE, 'api.proto', '--go_out=plugins=grpc:.', cwd=API_PACKAGE)


@command()
def executable():
    # _call('go', 'build', '-o', FINAL_EXECUTABLE, cwd=MAIN_PACKAGE)
    _call('build/env.sh', 'go', 'run', 'build/ci.go', 'install', './cmd/evm', cwd=ROOT_DIR)


@command()
def all():
    clean_tmp()
    install_protoc()
    grpc_compile()
    executable()
