from ast import literal_eval, parse, Call
import inspect
from typing import *


class Shell:

    def __init__(self, **commands):
        self._commands = commands

    def command(self, fn: Callable, name: str = None):
        name = name or fn.__name__
        assert name and name not in self._commands
        self._commands[name] = fn
        return fn

    def meta(self) -> Dict[str, inspect.Signature]:
        return {name: inspect.signature(fn) for name, fn in self._commands.items()}

    def dump(self) -> str:
        return '\n'.join(name + str(inspect.signature(fn)) for name, fn in self._commands.items())

    def execute(self, call_notation: str) -> Any:
        ast = parse(call_notation)
        call_node = ast.body[0].value
        assert isinstance(call_node, Call)
        callee = self._commands[call_node.func.id]
        args = [literal_eval(node) for node in call_node.args]
        kwargs = {kwd.arg: literal_eval(kwd.value) for kwd in call_node.keywords}
        return callee(*args, **kwargs)

    def run_cli(self):
        import sys
        args = sys.argv[1:]
        if not args:
            print(self.dump())
        elif len(args) == 1:
            ret = self.execute(args[0])
            if ret is not None:
                print(ret)
        else:
            raise RuntimeError('Too many arguments')
