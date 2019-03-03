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

#include "grpcClient.h"
#include "grpcServer.h"

using namespace std;
using namespace taraxa::vm;

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerReader;
using grpc::ServerReaderWriter;
using grpc::ServerWriter;
using grpc::Status;
using grpc::ClientContext;

using statedb::BytesMessage;
using statedb::BoolMessage;
using statedb::VmId;
using statedb::StateDB;

void DoTest() {
    grpcClient client(grpc::CreateChannel(
            "0.0.0.0:50051", grpc::InsecureChannelCredentials()));

    ::statedb::BytesMessage request;
    request.set_value("1234567890");
    auto vmid = request.mutable_vmid();
    vmid->set_contractaddr("0987654321");
    vmid->set_processid("999");

    Status s = client.Put(request);
    cout << "Put " << request.value() << " ,vmid " << request.vmid().processid() << "," << request.vmid().contractaddr();
    cout << boolalpha << " ,Status: " << s.ok() << endl;
    EXPECT_TRUE(s.ok());

    BoolMessage has_responce;
    has_responce = client.Has(request);
    cout << boolalpha << "Has " << has_responce.value() << " ,vmid " << request.vmid().processid() << "," << request.vmid().contractaddr() << endl;
    EXPECT_TRUE(has_responce.value());

    BytesMessage get_responce;
    get_responce = client.Get(request);
    cout << "Get " << get_responce.value() << " ,vmid " << request.vmid().processid() << "," << request.vmid().contractaddr() << endl;
    EXPECT_TRUE(get_responce.value() == "1234567890");

    s = client.Delete(request);
    cout << boolalpha << "Delete " << request.value() << " ,Status: " << s.ok() << endl;
    EXPECT_TRUE(s.ok());

    has_responce = client.Has(request);
    cout << boolalpha << "Has " << has_responce.value() << " ,vmid " << request.vmid().processid() << "," << request.vmid().contractaddr() << endl;
    EXPECT_FALSE(has_responce.value());
}

TEST(DoTest, SimpleRpc) {
    DoTest();
}

#endif //TESTS_INTEGRATION_GRPCINTEGRITY_H
