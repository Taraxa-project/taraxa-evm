#ifndef TARAXA_EVM_CTYPES_H
#define TARAXA_EVM_CTYPES_H

#include <stdlib.h>
#include <stdint.h>

typedef long long GoInt;
#define GO_SLICE(type) struct { type *Data; GoInt Len; GoInt Cap; }

#define DEFINE_CALLBACK(type) \
    typedef struct { void *receiver; void (*apply)(void *, type *); } type##Callback; \
    inline void type##CallbackApply(type##Callback cb, type *arg) { if (cb.apply) cb.apply(cb.receiver, arg); } \

typedef uint8_t taraxa_evm_Address[20];
typedef uint8_t taraxa_evm_Hash[32];
typedef GO_SLICE(taraxa_evm_Hash) taraxa_evm_Hashes;

typedef struct {
    GO_SLICE(uint8_t) Value;
    GO_SLICE(GO_SLICE(uint8_t)) Nodes;
} taraxa_evm_trie_Proof;
typedef struct {
    taraxa_evm_trie_Proof AccountProof;
    GO_SLICE(taraxa_evm_trie_Proof) StorageProofs;
} taraxa_evm_state_Proof;
DEFINE_CALLBACK(taraxa_evm_state_Proof);

#undef DEFINE_CALLBACK

#endif