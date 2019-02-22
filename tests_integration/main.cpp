#include <iostream>
#include <gtest/gtest.h>
#include "include/Process.h"
#include "include/ProcessTest.h"
#include <rapidjson/document.h>

using namespace std;
using namespace rapidjson;

TEST(TestBlindAuction, TestBlindAuction_Positive_Test) {
    ASSERT_STREQ("", TestBlindAuction().c_str());
}

TEST(TestAcknowledger, TestAcknowledger_Positive_Test) {
    ASSERT_STREQ("", TestAcknowledger().c_str());
}

TEST(TestCodeVerify, TestCodeVerify_Negative_Test) {
    ASSERT_STRNE("", TestCodeVerify().c_str());
}

TEST(TestVulnerabilityRecursive, TestVulnerabilityRecursive_Negative_Test) {
    ASSERT_STRNE("", TestVulnerabilityRecursive().c_str());
}

TEST(TestVulnerabilityLoop, TestVulnerabilityLoop_Negative_Test) {
    ASSERT_STRNE("", TestVulnerabilityLoop().c_str());
}

TEST(TestGasOK, TestGasOK_Positive_Test) {
    ASSERT_STREQ("", TestGasOK().c_str());
}

TEST(TestOutOfGas, TestOutOfGas_Negative_Test) {
    ASSERT_STRNE("", TestOutOfGas().c_str());
}

int main(int argc, char** argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
