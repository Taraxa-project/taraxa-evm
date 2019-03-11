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

namespace taraxa::__StateDBClient {
    using namespace std;
    using namespace taraxa::vm::statedb;
    using namespace google::protobuf;
    using namespace grpc;

    class StateDBClient {

        unique_ptr<StateDB::Stub> grpcClient;

    public:

        explicit StateDBClient(const shared_ptr<Channel> &channel) : grpcClient(StateDB::NewStub(channel)) {}

        Status Put(const KeyAndValueMessage &message) {
            Empty response;
            ClientContext context;
            Status status = grpcClient->Put(&context, message, &response);
            return status;
        }

        Status Delete(const KeyMessage &message) {
            Empty response;
            ClientContext context;
            Status status = grpcClient->Delete(&context, message, &response);
            return status;
        }

        BytesMessage Get(const KeyMessage &message) {
            BytesMessage response;
            ClientContext context;
            Status status = grpcClient->Get(&context, message, &response);
            return response;
        }

        BoolMessage Has(const KeyMessage &message) {
            BoolMessage response;
            ClientContext context;
            grpcClient->Has(&context, message, &response);
            return response;
        }

        Status Close(const VmId &message) {
            Empty response;
            ClientContext context;
            return grpcClient->Close(&context, message, &response);
        }

    };

}
namespace taraxa {
    using __StateDBClient::StateDBClient;
}

#endif //TESTS_INTEGRATION_GRPCCLIENT_H
