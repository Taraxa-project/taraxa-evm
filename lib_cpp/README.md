## C++ bindings library for the evm golang package

- [(SUSPENDED) Smart contract utils](contracts/README.md)

### GLOSSARY
- THIS_DIR - path to this very directory (containing this README)

### Requirements
- [Main requirements](../README.md#Requirements)
- cmake 3.13+
- C++17-compatible compiler

The library has header dependencies on `rapidjson` and `boost`.
It installs them on it's own: downloads from github and builds from source.
It's straightforward to use the library in Cmake builds: just add this directory as
Cmake subdirectory, and link against `taraxa_evm` library target. It will add header dependencies
automatically.
For non-cmake builds it's required to add the `rapidjson/*` and `boost/*` headers to the header 
search path. 
You can use your own versions of those, or reuse the paths installed by the library:
- `THIS_DIR/thirdparty/boost-cmake-src/boost/boost_1_67_0`
- `THIS_DIR/thirdparty/rapidjson-project-src/include`

### Building
- Choose *any* build directory, let's call it BUILD_DIR
- From within BUILD_DIR: 
`cmake THIS_DIR` 
and then 
`cmake --build . --target taraxa_evm -j <number of processors for parallel build>`

### Testing
From within BUILD_DIR: 
- `cmake --build . --target taraxa_evm_tests -j <number of processors for parallel build>`
- `BUILD_DIR/taraxa_evm_tests`

### Products
- `BUILD_DIR/libtaraxa_evm.a` - static library to link against
- `THIS_DIR/include` - project headers root
- `taraxa_evm/lib.hpp` - the main header
- `taraxa_evm::lib::runEvm` - the function to run the EVM