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

namespace taraxa::__StateDBMockServer {
    using namespace std;
    using namespace taraxa::vm::statedb;
    using namespace google::protobuf;
    using namespace grpc;

    class StateDBMockServer final : public StateDB::Service {

        map<string, BytesMessage> messages;

        static string toStr(const VmId &vmid) {
            return vmid.processid() + "_" + vmid.contractaddress();
        }

        static string toStr(const KeyMessage &keyMessage) {
            return toStr(keyMessage.vmid()) + "_" + keyMessage.memoryaddress().value();
        }

    public:

        Status Put(ServerContext *context, const KeyAndValueMessage *request, Empty *response) {
            auto keyStr = toStr(request->key());
            auto value = request->value();
            cout << "RPC PUT key: " << keyStr << ", value: " << value.value() << endl;
            messages.emplace(keyStr, value);
            return Status::OK;
        }

        Status Delete(ServerContext *context, const KeyMessage *request, Empty *response) {
            auto keyStr = toStr(*request);
            cout << "RPC DELETE key: " << keyStr << endl;
            messages.erase(keyStr);
            return Status::OK;
        };

        Status Get(ServerContext *context, const KeyMessage *request, BytesMessage *response) {
            auto keyStr = toStr(*request);
            auto entry = messages.find(keyStr);
            if (entry != messages.end()) {
                response->CopyFrom(entry->second);
            }
            cout << "RPC GET key: " << keyStr << ", value : " << response->value() << endl;
            return Status::OK;
        };

        Status Has(ServerContext *context, const KeyMessage *request, BoolMessage *response) {
            auto keyStr = toStr(*request);
            response->set_value(messages.find(keyStr) != messages.end());
            cout << boolalpha << "RPC HAS key: " << keyStr << ", value : " << response->value() << endl;
            return Status::OK;
        };

        Status Close(ServerContext *context, const VmId *request, Empty *response) {
            cout << "RPC CLOSE vmid: " << toStr(*request) << endl;
            return Status::OK;
        };

    };

}
namespace taraxa {
    using __StateDBMockServer::StateDBMockServer;
}

#endif //TESTS_INTEGRATION_GRPCSERVER_H
