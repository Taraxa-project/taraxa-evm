#include <stdio.h>
#include <stdlib.h>

//void foo() {}
#include "ctypes.h"
#include "taraxa-evm.h"

void on_proof(void* receiver, taraxa_evm_state_Proof* v) {
    printf("ololo\n");
//    printf(v);
}

typedef struct {
    taraxa_evm_Hash h;
} HashWrap;

HashWrap get_hash() {
    HashWrap ret = {0};
    ret.h[0] = 66;
    ret.h[20] = 32;
    return ret;
}

void foo() {
    printf("foo\n");
    HashWrap root = get_hash();
    taraxa_evm_Address addr = {0};
    addr[0] = 2;
    taraxa_evm_Hashes hashes = {&root.h, 1, 1};
    taraxa_evm_state_ProofCallback cb = {0};
    cb.apply = on_proof;
    taraxa_evm_state_Prove(32, &root.h, &addr, hashes, cb);
}


void test_main() {
    foo();
}