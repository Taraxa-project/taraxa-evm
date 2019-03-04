#ifndef TESTS_INTEGRATION_GRPCINTEGRITY_H
#define TESTS_INTEGRATION_GRPCINTEGRITY_H

#include <iostream>
#include <thread>

#include <grpc/grpc.h>
#include <grpc/support/alloc.h>
#include <grpc/support/log.h>
#include <grpc/support/time.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/ext/health_check_service_server_builder_option.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include "include/StateDBClient.h"
#include "include/StateDBMockServer.h"
#include "include/grpc_util.hpp"

using namespace std;
using namespace taraxa::vm;
using namespace statedb;

TEST(DoTest, SimpleRpc) {
    StateDBMockServer stateDBMockServer;
    auto serverAddress = "0.0.0.0:50051";
    auto server = startGRPCService(&stateDBMockServer, serverAddress);
    StateDBClient client(grpc::CreateChannel(serverAddress, grpc::InsecureChannelCredentials()));
    thread serverThread([&]() {
        cout << "Running. Server listening on " << serverAddress << std::endl;
        server->Wait();
        cout << "Server at " << serverAddress << " has stopped." << std::endl;
    });

    KeyAndValueMessage putRequest;
    putRequest.mutable_value()->set_value("1234567890");
    auto key = putRequest.mutable_key();
    auto vmid = key->mutable_vmid();
    vmid->set_contractaddress("0987654321");
    vmid->set_processid("999");
    key->mutable_memoryaddress()->set_value("234231");

    EXPECT_TRUE(client.Put(putRequest).ok());
    EXPECT_TRUE(client.Has(putRequest.key()).value());
    EXPECT_TRUE(client.Get(putRequest.key()).value() == putRequest.value().value());
    EXPECT_TRUE(client.Delete(putRequest.key()).ok());
    EXPECT_FALSE(client.Has(putRequest.key()).value());

    server->Shutdown();
    serverThread.join();
}

#endif //TESTS_INTEGRATION_GRPCINTEGRITY_H
