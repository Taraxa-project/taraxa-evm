#include <iostream>
#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "include/Process.h"

using namespace std;
using namespace rapidjson;

TEST(EVM, TestBlindAuction_Positive_Test) {
    ASSERT_EQ("", RunCodeFile("clear_contracts/BlindAuction.bin"));
}

TEST(EVM, TestAcknowledger_Positive_Test) {
    ASSERT_EQ("", RunCodeFile("clear_contracts/Acknowledger.bin"));
}

TEST(EVM, TestGasOK_Positive_Test) {
    ASSERT_EQ("", RunTest("--code 6040 --json run"));
}

TEST(EVM, TestCodeVerify_Negative_Test) {
    ASSERT_NE("", RunCodeFile("crash_contracts/unverify.bin"));
}

TEST(EVM, TestVulnerabilityRecursive_Negative_Test) {
    ASSERT_NE("", RunCodeFile("crash_contracts/recursive.bin"));
}

TEST(EVM, TestVulnerabilityLoop_Negative_Test) {
    ASSERT_NE("", RunCodeFile("crash_contracts/loop.bin"));
}

TEST(EVM, TestOutOfGas_Negative_Test) {
    ASSERT_NE("", RunTest("--code 6040 --gas 0x1 --price 0x3 --json run"));
}