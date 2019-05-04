import inspect
from contextlib import ExitStack
from functools import wraps


def with_exit_stack(fn):
    @wraps(fn)
    def wrapper(*args, **kwargs):
        with ExitStack() as ___exit_stack___:
            return fn(*args, **kwargs)

    return wrapper


def current_exit_stack() -> ExitStack:
    current_frame = inspect.currentframe()
    caller_caller_frame = current_frame.f_back.f_back
    return caller_caller_frame.f_locals.get('___exit_stack___', None)
