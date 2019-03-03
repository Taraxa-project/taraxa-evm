//
// Created by ibox on 3/3/19.
//

#include <iostream>
#include <gtest/gtest.h>

#include "grpcIntegrity.h"

using namespace std;

int main(int argc, char** argv) {
    testing::InitGoogleTest(&argc, argv);

    auto server_ = Start();

    thread task(RunServer, server_);
    cout << boolalpha << task.get_id() << " joinable " << task.joinable() << endl;

    //server_->Shutdown();
    //task.join();

    return RUN_ALL_TESTS();
}
