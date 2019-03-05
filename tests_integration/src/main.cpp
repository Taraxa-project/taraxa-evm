#include <gtest/gtest.h>

#include "include/tests/mock_grpc_server.hpp"
#include "include/tests/evm_cli_mem_db.hpp"

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}