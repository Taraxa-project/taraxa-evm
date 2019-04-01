#include <gtest/gtest.h>
#include "taraxa_evm/util_str.hpp"
#include "taraxa_evm/util_io.hpp"
#include "taraxa_evm/paths.hpp"
#include "taraxa_evm/lib.hpp"

using namespace std;
using namespace taraxa_evm;
using namespace taraxa_evm::lib;
using namespace taraxa_evm::util_io;

TEST(typed_api, create_contract) {
    LDBConfig ldbConfig{
            createFreshTmpDir("test_typed_api_create_contract"),
            0,
            0
    };
    Block block{
            "0x0000000000000000000000000000000000000064",
            0,
            0,
            0,
            100000000000,
            "0x0000000000000000000000000000000000000000000000000000000000000000",
    };
    string contractCode = "0x6080604052600560005534801561001557600080fd5b5060a2806100246000396000f3fe6080604052348015"
                          "600f57600080fd5b506004361060325760003560e01c806360fe47b11460375780636d4ce63c146053575b6000"
                          "80fd5b605160048036036020811015604b57600080fd5b5035606b565b005b60596070565b6040805191825251"
                          "9081900360200190f35b600055565b6000549056fea165627a7a723058202077afb60be630674b4b25a1657ffe"
                          "4edea0ac59876c8ca6aaffd5b3f22379330029";
    vector<Transaction> transactions{
            Transaction{
                    "0x0000000000000000000000000000000000000064",
                    nullptr,
                    0,
                    0,
                    100000000,
                    0,
                    &contractCode,
            }
    };
    auto result = runEvm(RunConfiguration{
            "0x0000000000000000000000000000000000000000000000000000000000000000",
            &block,
            &transactions,
            &ldbConfig,
            nullptr
    }, ExternalApi{
            [](auto i) {
                return "fooo";
            }
    });
    ASSERT_EQ(result.stateRoot, "0xf274095ef03a004be6214d9b05baa1c3051f7276b5e106af21cc73e282f09148");
    ASSERT_EQ(result.receipts[0].contractAddress, "0x86c56c43a1d19b06d54971c467bad4b25e4ef59e");
    ASSERT_EQ(result.concurrentSchedule.sequential.size(), 0);
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}