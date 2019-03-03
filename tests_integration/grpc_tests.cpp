//
// Created by ibox on 3/3/19.
//

#include <iostream>
#include <gtest/gtest.h>

#include "grpcIntegrity.h"

using namespace std;

int main(int argc, char** argv) {
    testing::InitGoogleTest(&argc, argv);

    thread task(RunServer);
    task.join();

    return RUN_ALL_TESTS();
}
