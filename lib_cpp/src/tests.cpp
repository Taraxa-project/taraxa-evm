#include <gtest/gtest.h>
#include <boost/filesystem.hpp>
#include "taraxa_evm/util_str.hpp"
#include "taraxa_evm/paths.hpp"
#include "taraxa_evm/lib.hpp"

#include <iostream>

using namespace std;
using namespace boost::filesystem;
using namespace taraxa_evm;
using namespace taraxa_evm::lib;

TEST(integration, json_api) {
    auto ldbPath = temp_directory_path() / "test_integration_json_api_ldb";
    if (is_directory(ldbPath)) {
        remove_all(ldbPath);
    }
    create_directory(ldbPath);
    auto config = util_str::fmt(R"(
{
    "stateRoot": "0x0b12df123a6162a1e53d1d4ca9263a2e5129d0f047646d36fc9cdd56dc978172",
    "block": {
        "coinbase": "0x0000000000000000000000000000000000000064",
        "number": 0,
        "time": 0,
        "difficulty": 0,
        "gasLimit": 100000000000,
        "hash": "0x0000000000000000000000000000000000000000000000000000000000000000"
    },
    "transactions": [
        {
            "to": "0x86c56c43a1d19b06d54971c467bad4b25e4ef59e",
            "from": "0x0000000000000000000000000000000000000066",
            "nonce": 0,
            "amount": 0,
            "gasLimit": 100000000,
            "gasPrice": 0,
            "data": "YP5HsQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC"
        },
        {
            "to": "0x208dae7e1aafbdde0aac383bd2b7868dab4c37ac",
            "from": "0x0000000000000000000000000000000000000067",
            "nonce": 0,
            "amount": 0,
            "gasLimit": 100000000,
            "gasPrice": 0,
            "data": "YP5HsQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
        },
        {
            "to": "0x208dae7e1aafbdde0aac383bd2b7868dab4c37ac",
            "from": "0x0000000000000000000000000000000000000067",
            "nonce": 1,
            "amount": 0,
            "gasLimit": 100000000,
            "gasPrice": 0,
            "data": "bUzmPA=="
        }
    ],
    "ldbConfig": {
        "file": "%s",
        "cache": 0,
        "handles": 0
    },
    "concurrentSchedule": null
}
    )", ldbPath.string());
    auto expectedResult = R"({"stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","concurrentSchedule":null,"receipts":null,"allLogs":null,"usedGas":0,"returnValues":null,"error":{"NodeHash":"0x0b12df123a6162a1e53d1d4ca9263a2e5129d0f047646d36fc9cdd56dc978172","Path":null}})";
    auto result = cgo_bridge::run(config);
    ASSERT_STREQ(expectedResult, result.c_str());
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}