// server example
#ifndef TESTS_INTEGRATION_GRPCSERVER_H
#define TESTS_INTEGRATION_GRPCSERVER_H

#include <algorithm>
#include <iostream>
#include <memory>
#include <string>

#include <grpc/grpc.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>
#include <grpcpp/security/server_credentials.h>

#include "common.pb.h"
#include "statedb.pb.h"
#include "statedb.grpc.pb.h"

using namespace std;
using namespace taraxa::vm;

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerReader;
using grpc::ServerReaderWriter;
using grpc::ServerWriter;
using grpc::Status;

using statedb::BytesMessage;
using statedb::BoolMessage;
using statedb::VmId;
using statedb::StateDB;

class StateDBMockServer final : public StateDB::Service {

private:

    std::map<std::string, ::statedb::BytesMessage> messages;

public:

    Status Put(ServerContext *context,
               const ::statedb::KeyAndValueMessage *request,
               ::google::protobuf::Empty *response) {
        auto key_str = to_str(request->key());
        auto value = request->value();
        cout << "PUT key: " << key_str << ", value: " << value.value() << endl;
        messages.emplace(key_str, value);
        return Status::OK;
    }

    Status Delete(::grpc::ServerContext *context,
                  const ::taraxa::vm::statedb::KeyMessage *request,
                  ::google::protobuf::Empty *response) {
        auto key_str = to_str(*request);
        cout << "DELETE key: " << key_str << endl;
        messages.erase(key_str);
        return Status::OK;
    };

    Status Get(::grpc::ServerContext *context,
               const ::taraxa::vm::statedb::KeyMessage *request,
               ::taraxa::vm::statedb::BytesMessage *response) {
        auto key_str = to_str(*request);
        auto entry = messages.find(key_str);
        if (entry != messages.end()) {
            response->CopyFrom(entry->second);
        }
        cout << "GET key: " << key_str << ", value : " << response->value() << endl;
        return Status::OK;
    };

    Status Has(::grpc::ServerContext *context,
               const ::taraxa::vm::statedb::KeyMessage *request,
               ::taraxa::vm::statedb::BoolMessage *response) {
        auto key_str = to_str(*request);
        response->set_value(messages.find(key_str) != messages.end());
        cout << boolalpha << "HAS key: " << key_str << ", value : " << response->value() << endl;
        return Status::OK;
    };

    Status Close(::grpc::ServerContext *context,
                 const ::taraxa::vm::statedb::VmId *request,
                 ::google::protobuf::Empty *response) {
        cout << "CLOSE vmid: " << to_str(*request) << endl;
        return Status::OK;
    };

private:

    static std::string to_str(const ::statedb::VmId &vmid) {
        return vmid.processid() + "_" + vmid.contractaddress();
    }

    static std::string to_str(const ::statedb::KeyMessage &keyMessage) {
        return to_str(keyMessage.vmid()) + "_" + keyMessage.memoryaddress().value();
    }

};

#endif //TESTS_INTEGRATION_GRPCSERVER_H
