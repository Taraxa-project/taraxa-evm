#ifndef TARAXA_EVM_CGO_IMPORTS_C
#define TARAXA_EVM_CGO_IMPORTS_C

#include "taraxa_evm/cgo_imports.h"

const char *getHeaderHashByBlockNumber(ExternalApi *api, uint64_t val) {
    return api->getHeaderHashByBlockNumber(val);
}

#endif //TARAXA_EVM_CGO_IMPORTS_C
