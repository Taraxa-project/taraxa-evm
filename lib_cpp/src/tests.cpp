#include <gtest/gtest.h>
#include "taraxa_evm/util_str.hpp"
#include "taraxa_evm/util_io.hpp"
#include "taraxa_evm/paths.hpp"
#include "taraxa_evm/lib.hpp"

#include <iostream>

using namespace std;
using namespace taraxa_evm;
using namespace taraxa_evm::lib;
using namespace taraxa_evm::util_io;

TEST(typed_api, create_contract) {
    LDBConfig ldbConfig{
            .file = createFreshTmpDir("test_typed_api_create_contract")
    };
    Block block{
            .gasLimit = 100000000000,
            .difficulty = 0,
            .time = 0,
            .number = 0,
            .hash = "0x0000000000000000000000000000000000000000000000000000000000000000",
            .coinbase = "0x0000000000000000000000000000000000000064",
    };
    string contractCode = "YIBgQFJgBWAAVTSAFWEAFVdgAID9W1BgooBhACRgADlgAPP+YIBgQFI0gBVgD1dgAID9W1BgBDYQYDJXYAA1YOAcgG"
                          "Ng/kexFGA3V4BjbUzmPBRgU1dbYACA/VtgUWAEgDYDYCCBEBVgS1dgAID9W1A1YGtWWwBbYFlgcFZbYECAUZGCUlGQ"
                          "gZADYCABkPNbYABVVltgAFSQVv6hZWJ6enIwWCAgd6+2C+YwZ0tLJaFlf/5O3qCsWYdsjKaq/9Wz8iN5MwAp";
    vector<Transaction> transactions{
            Transaction{
                    .nonce = 0,
                    .from = "0x0000000000000000000000000000000000000064",
                    .to = nullptr,
                    .data = &contractCode,
                    .amount= 0,
                    .gasPrice= 0,
                    .gasLimit= 100000000,
            }
    };
    auto result = runEvm(RunConfiguration{
            .stateRoot = "0x0000000000000000000000000000000000000000000000000000000000000000",
            .block = &block,
            .transactions = &transactions,
            .ldbConfig = &ldbConfig,
            .concurrentSchedule = nullptr
    });
    ASSERT_EQ(result.stateRoot, "0xf274095ef03a004be6214d9b05baa1c3051f7276b5e106af21cc73e282f09148");
    ASSERT_EQ(result.receipts[0].contractAddress, "0x86c56c43a1d19b06d54971c467bad4b25e4ef59e");
    ASSERT_EQ(result.concurrentSchedule.sequential.size(), 0);
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}