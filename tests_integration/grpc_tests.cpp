#include <iostream>

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include "grpcIntegrity.h"

using namespace std;

int main(int argc, char** argv) {
    ::testing::InitGoogleTest(&argc, argv);

    return RUN_ALL_TESTS();
}
