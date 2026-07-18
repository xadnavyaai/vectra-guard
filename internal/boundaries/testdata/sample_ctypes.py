# Sample Python file with ctypes FFI boundaries.

import ctypes
from cffi import FFI

libc = ctypes.CDLL("libc.so.6")


def safe_add(a, b):
    return a + b
