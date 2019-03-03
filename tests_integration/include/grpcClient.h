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
    grpcClient(std::shared_ptr<Channel> channel, const std::string& db)
    : stub_(StateDB::NewStub(channel)) {
    }

};


#endif //TESTS_INTEGRATION_GRPCCLIENT_H
