import os
import platform
import shutil
import sys
from os import path
from os.path import join
from pathlib import Path
from subprocess import call

PROJECT_NAME = 'go-ethereum'

THIS_DIR = Path(__file__).parent.absolute()
ROOT_DIR = Path(THIS_DIR).parent
DEV_DEPS_DIR = Path(THIS_DIR, 'dependencies')
PIP_DEPS_DIR = Path(DEV_DEPS_DIR, 'pip')
PROTOC_DIR = Path(DEV_DEPS_DIR, 'protoc')
PROTOC_EXECUTABLE = Path(PROTOC_DIR, 'bin', 'protoc')
BIN_DIR = Path(THIS_DIR, 'bin')
GRPC_ROOT = Path(ROOT_DIR, 'grpc')
PROTOBUF_ROOT = Path(GRPC_ROOT, 'protobuf')
GRPC_GO_PACKAGE = Path(GRPC_ROOT, 'grpc_go')
GRPC_CPP_PACKAGE = Path(GRPC_ROOT, 'grpc_cpp')
CMD_DIR = Path(ROOT_DIR).joinpath('cmd')
EVM_CMD_DIR = Path(CMD_DIR, 'evm')
FINAL_EXECUTABLE = Path(BIN_DIR, 'evm')

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
    for line in open(join(ROOT_DIR, '.tmp_files')).readlines():
        path_pattern = line.split('# ')[0].strip()
        if path_pattern.startswith('/'):
            path_pattern = path_pattern[1:]
        if not path_pattern or path_pattern == '#':
            continue
        for p in Path(ROOT_DIR).glob(path_pattern):
            path_str = str(p.absolute())
            print(f'Removing {path_str}')
            if p.is_dir():
                shutil.rmtree(path_str)
            else:
                os.remove(path_str)


@command()
def install_protoc():
    from io import BytesIO
    from zipfile import ZipFile
    from urllib.request import urlopen
    import stat
    # TODO more platforms
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
    print(f'Installed protoc to {PROTOC_DIR}')


@command()
def grpc_compile():
    print(f'Compiling .proto files at {PROTOBUF_ROOT}')
    for proto_file in Path(PROTOBUF_ROOT).glob("**/*.proto"):
        proto_package = proto_file.parent
        rel_package = proto_package.relative_to(PROTOBUF_ROOT)
        go_package, cpp_package = (Path(grpc_package).joinpath(rel_package).absolute()
                                   for grpc_package in (GRPC_GO_PACKAGE, GRPC_CPP_PACKAGE))
        for dir in go_package, cpp_package:
            if not dir.exists():
                dir.mkdir(parents=True)
        # TODO NOT READY
        _call(PROTOC_EXECUTABLE, proto_file.name,
              f'--go_out=plugins=grpc:{go_package}',
              f'--cpp_out={cpp_package}',
              f'--grpc_out={cpp_package}',
              f'--plugin=protoc-gen-grpc=TODO',
              f'--proto_path={PROTOBUF_ROOT}',
              cwd=str(proto_package))


@command()
def executable():
    print(f'Building evm')
    # _call('go', 'build', '-o', FINAL_EXECUTABLE, cwd=EVM_CMD_DIR)
    _call('go', 'run', 'build/ci.go', 'install', './cmd/evm', cwd=ROOT_DIR)
    print(f'Installed evm executable to {BIN_DIR}')


@command()
def all():
    clean_tmp()
    install_protoc()
    grpc_compile()
    executable()
