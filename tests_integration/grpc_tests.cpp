#include <iostream>

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include "grpcIntegrity.h"

using namespace std;

int main(int argc, char** argv) {
    ::testing::InitGoogleTest(&argc, argv);

    auto server_ = Start();
    thread task(RunServer, server_);
    cout << boolalpha << task.get_id() << " joinable " << task.joinable() << endl;
    task.join();
    return RUN_ALL_TESTS();
}
