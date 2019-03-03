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

// auto generate by protoc -I . --grpc_out=generate_mock_code=true:. --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` statedb.proto
//#include "statedb_mock.grpc.pb.h"
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

class grpcIntegrity : public grpcClient {
public:
    grpcIntegrity(std::shared_ptr<Channel> channel)
    :grpcClient(channel)
    {}

    void DoTest() {
        cout << "1" << endl;
        ::google::protobuf::Empty response;
        ClientContext context;
        ::statedb::BytesMessage request;
        request.set_value("1234567890");
        auto vmid = request.mutable_vmid();
        vmid->set_contractaddr("0987654321");
        vmid->set_processid("999");

        Status s = stub_->Put(&context, request, &response);
        EXPECT_TRUE(s.ok());

        BoolMessage has_responce;
        s = stub_->Has(&context, request, &has_responce);
        EXPECT_TRUE(has_responce.value());

        BytesMessage get_responce;
        s = stub_->Get(&context, request, &get_responce);
        EXPECT_TRUE(s.ok());
        EXPECT_TRUE(get_responce.value() == "1234567890");

        s = stub_->Delete(&context, request, &response);
        EXPECT_TRUE(s.ok());

        s = stub_->Has(&context, request, &has_responce);
        EXPECT_FALSE(has_responce.value());

    }

};

void DoTest() {
    grpcIntegrity client(grpc::CreateChannel(
            "localhost:50051", grpc::InsecureChannelCredentials()));
    client.DoTest();
}

TEST(DoTest, SimpleRpc) {
    DoTest();
}

#endif //TESTS_INTEGRATION_GRPCINTEGRITY_H
