import json
from ctypes import cdll, c_char_p

from paths import *
from .util import call


class TaraxaEvm:
    go_library_package = base_dir.parent.joinpath('main')

    def __init__(self, library_path):
        lib = cdll.LoadLibrary(library_path)
        lib.RunTaraxaEvm.argtypes = [c_char_p]
        lib.RunTaraxaEvm.restype = c_char_p
        self._lib = lib

    def run(self, request):
        json_bytes = json.dumps(request).encode()
        ret_json_bytes = self._lib.RunTaraxaEvm(json_bytes)
        return json.loads(ret_json_bytes)

    @classmethod
    def build_c_lib(cls, output_path):
        call(f"go build -tags=lib_cpp -buildmode=c-shared -o {output_path}", cwd=cls.go_library_package)
