import json
from abc import ABC
from contextlib import contextmanager
from ctypes import cdll, c_char_p
from paths import *
from taraxa.util import call

LOCAL_LIB_PACKAGE = base_dir.parent.joinpath('main')


class Callable(ABC):

    def call(self, method_name: str, *args) -> list:
        raise NotImplementedError()


class Pointer(Callable, ABC):

    def free(self):
        pass

    @contextmanager
    def scope(self):
        try:
            yield self
        finally:
            self.free()


class LibTaraxaEvm(Callable):

    def __init__(self, library_path):
        lib = cdll.LoadLibrary(library_path)
        lib.Call.argtypes = [c_char_p, c_char_p, c_char_p]
        lib.Call.restype = c_char_p
        lib.Free.argtypes = [c_char_p]
        self._lib = lib

    @staticmethod
    def build(output_path, package_path=LOCAL_LIB_PACKAGE):
        call(f"go build -tags=lib_cpp -buildmode=c-shared -o {output_path} {package_path}")  # , cwd=path)

    def as_ptr(self, addr: str) -> Pointer:
        self_ = self

        class PointerImpl(Pointer):

            def free(self):
                self_._lib.Free(addr.encode())

            def call(self, method_name: str, *args) -> list:
                return self_._call(addr, method_name, *args)

        return PointerImpl()

    def call(self, function_name, *args):
        return self._call("", function_name, *args)

    def _call(self, receiver_addr: str, function_name: str, *args) -> list:
        args_str = json.dumps(args)
        # print(f'lib_taraxa_vm call: {receiver_addr}.{function_name}({args_str})')
        ret_encoded = self._lib.Call(receiver_addr.encode(), function_name.encode(), args_str.encode())
        # print(f'lib_taraxa_vm result: {str(ret_encoded, encoding="utf-8")}')
        return json.loads(ret_encoded)
