#ifndef TARAXA_EVM_COMMON_H
#define TARAXA_EVM_COMMON_H

#include <stdint.h>

#define SLICE(type) struct { type *Data; size_t Len; }
#define FUNCTION(name, in_t, out_t) \
    typedef struct {  \
        void *Receiver; \
        out_t (*Apply)(void *, in_t); \
    } name; \
    inline out_t name##Apply(name fn, in_t arg) { return fn.Apply(fn.Receiver, arg); } \

typedef SLICE(uint8_t) taraxa_evm_Bytes;
typedef struct {
    uint8_t Val[32];
} taraxa_evm_Hash;

FUNCTION(taraxa_evm_BytesCallback, taraxa_evm_Bytes, void);
FUNCTION(taraxa_evm_GetBlockHash, uint64_t, taraxa_evm_Hash);

#undef SLICE
#undef FUNCTION

#endif