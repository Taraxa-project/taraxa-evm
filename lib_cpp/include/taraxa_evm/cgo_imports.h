#ifndef TARAXA_EVM_TARAXA_EVM_CGO_IMPORTS_H
#define TARAXA_EVM_TARAXA_EVM_CGO_IMPORTS_H

#include <stdint.h>

typedef struct {
    const char *(*getHeaderHashByBlockNumber)(uint64_t);
} ExternalApi;

const char *getHeaderHashByBlockNumber(ExternalApi *api, uint64_t val);

#endif //TARAXA_EVM_TARAXA_EVM_CGO_IMPORTS_H