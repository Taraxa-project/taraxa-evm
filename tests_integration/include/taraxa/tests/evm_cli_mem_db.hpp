#include <iostream>
#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "taraxa/evm.hpp"
#include "taraxa/StateDBClient.hpp"
#include "taraxa/StateDBMockServer.hpp"
#include "taraxa/util_grpc.hpp"

namespace taraxa::tests::evm_cli_mem_db {
    using namespace std;
    using namespace rapidjson;
    using namespace taraxa::util_grpc;
    using namespace taraxa;
    using evm::JsonOutput;

    TEST(EVM_RPC_DB, create_contract_single_var_init_value) {
        StateDBMockServer stateDBMockServer;
        auto serverAddress = "0.0.0.0:50051";
        auto server = startGRPCService(&stateDBMockServer, serverAddress);
        auto code = contracts::getCode("SingleVariable");
        auto createResult = evm::runCode(
                code,
                "--statedb.address", serverAddress,
                "--create");
        auto setResult = evm::runCode(
                code,
                "--statedb.address", serverAddress,
                "--input", contracts::generateCall("SingleVariable", "set", "1"));
        auto getResult = evm::runCode(
                code,
                "--statedb.address", serverAddress,
                "--input", contracts::generateCall("SingleVariable", "get"));
        server->Shutdown();
        server->Wait();
    }

//    TEST(EVM, TestBlindAuction_Positive_Test) {
//        cout << "foo" << endl;
//        ASSERT_EQ(evmRunContract("BlindAuction").error, "");
//    }

    TEST(EVM, TestAcknowledger_Positive_Test) {
        auto output = evm::runCode(contracts::getCode("Acknowledger"));
        ASSERT_EQ(output.error, "");
    }

    TEST(EVM, TestGasOK_Positive_Test) {
        ASSERT_EQ(evm::runCode("6040").error, "");
    }

    TEST(EVM, TestOutOfGas_Negative_Test) {
        auto out = evm::runCode("6040", "--gas", "0x1", "--price", "0x3");
        ASSERT_EQ(out.error, "out of gas");
    }

//    TEST(EVM, TestCodeVerify_Negative_Test) {
//        ASSERT_NE(evmRunFile("crash_contracts/unverify.bin").error, "");
//    }
//
//    TEST(EVM, TestVulnerabilityRecursive_Negative_Test) {
//        ASSERT_NE(evmRunFile("crash_contracts/recursive.bin").error, "");
//    }
//
//    TEST(EVM, TestVulnerabilityLoop_Negative_Test) {
//        ASSERT_NE(evmRunFile("crash_contracts/loop.bin").error, "");
//    }

}