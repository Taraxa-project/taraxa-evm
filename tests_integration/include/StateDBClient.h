// client exmple
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

using namespace std;
using namespace taraxa::vm;

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

class StateDBClient {

    std::unique_ptr<StateDB::Stub> grpcClient;

public:

    explicit StateDBClient(const std::shared_ptr<Channel> &channel)
            : grpcClient(StateDB::NewStub(channel)) {
    }

    Status Put(const ::statedb::KeyAndValueMessage &message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        Status status = grpcClient->Put(&context, message, &response);
        return status;
    }

    Status Delete(const ::statedb::KeyMessage &message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        Status status = grpcClient->Delete(&context, message, &response);
        return status;
    }

    ::statedb::BytesMessage Get(const ::statedb::KeyMessage &message) {
        ::statedb::BytesMessage response;
        ClientContext context;
        Status status = grpcClient->Get(&context, message, &response);
        return response;
    }

    ::statedb::BoolMessage Has(const ::statedb::KeyMessage &message) {
        ::statedb::BoolMessage response;
        ClientContext context;
        grpcClient->Has(&context, message, &response);
        return response;
    }

    Status Close(const ::statedb::VmId &message) {
        ::google::protobuf::Empty response;
        ClientContext context;
        return grpcClient->Close(&context, message, &response);
    }

};


#endif //TESTS_INTEGRATION_GRPCCLIENT_H
