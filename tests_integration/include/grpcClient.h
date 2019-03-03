//
// Created by ibox on 3/3/19.
//

#ifndef TESTS_INTEGRATION_GRPCCLIENT_H
#define TESTS_INTEGRATION_GRPCCLIENT_H

#include <iostream>
#include <string>

#include <grpc/grpc.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/security/credentials.h>

#include "common.pb.h"
#include "statedb.pb.h"
#include "statedb.grpc.pb.h"

using namespace std
using namespace taraxa::vm

using grpc::Channel;
using grpc::ClientContext;
using grpc::ClientReader;
using grpc::ClientReaderWriter;
using grpc::ClientWriter;
using grpc::Status;

using statedb::VmId;
using statedb::BoolMessage;
using statedb::BytesMessage;
using statedb::StateDB;

class grpcClient {
public:
    grpcClient(std::shared_ptr<Channel> channel)
    : stub_(StateDB::NewStub(channel)) {
    }

    Status Put(const ::statedb::BytesMessage& message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        Status status = stub_->Put(&context, message, &response);
        return status;
    }

    Status Delete(const ::statedb::BytesMessage& message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        Status status = stub_->Delete(&context, message, &response);
        return status;
    }

    ::statedb::BytesMessage Get(const ::statedb::BytesMessage& message) {
        ::statedb::BytesMessage response;
        ClientContext context;
        Status status = stub_->Get(&context, message, &response);
        if (status != Status::OK)
            cout << "Error getter status" << endl;
        return response;
    }

    ::statedb::BoolMessage* Has(const ::statedb::BytesMessage& message) {
        ::statedb::BoolMessage response;
        ClientContext context;
        stub_->Has(&context, message, &response);
        return &response;
    }

    Status Close(const ::statedb::BytesMessage& message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        Status status = stub_->Close(&context, message, &response);
        return status;
    }

private:
    std::unique_ptr<StateDB::Stub> stub_;

};


#endif //TESTS_INTEGRATION_GRPCCLIENT_H
