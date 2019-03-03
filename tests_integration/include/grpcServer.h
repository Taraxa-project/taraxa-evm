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

class grpcServerImpl final : public StateDB::Service {

public:
    Status Put(ServerContext* context, const ::statedb::BytesMessage* request, ::google::protobuf::Empty* response) {
        if (!request->has_vmid())
            return Status::CANCELLED;
        cout << "PUT message " << request->value() << " ,vmid: " << request->vmid().processid() << "," << request->vmid().contractaddr() << endl;
        messages.emplace(getKeyFromVmId(request->vmid()), *request);
        return Status::OK;
    }
    Status Delete(::grpc::ServerContext* context, const ::taraxa::vm::statedb::BytesMessage* request, ::google::protobuf::Empty* response) {
        if (!request->has_vmid())
            return Status::CANCELLED;
        cout << "DELETE message ,vmid: " << request->vmid().processid() << "," << request->vmid().contractaddr() << endl;
        messages.erase(getKeyFromVmId(request->vmid()));
        return Status::OK;
    };
    Status Get(::grpc::ServerContext* context, const ::taraxa::vm::statedb::BytesMessage* request, ::taraxa::vm::statedb::BytesMessage* response) {
        if (!request->has_vmid())
            return Status::CANCELLED;
        auto it = messages.find(getKeyFromVmId(request->vmid()));
        if (it != messages.end()) {
            response->CopyFrom((*it).second);
        } else {
            return Status::CANCELLED;
        }
        cout << "GET message " << response->value() << " ,vmid: " << request->vmid().processid() << "," << request->vmid().contractaddr() << endl;
        return Status::OK;
    };
    Status Has(::grpc::ServerContext* context, const ::taraxa::vm::statedb::BytesMessage* request, ::taraxa::vm::statedb::BoolMessage* response) {
        if (!request->has_vmid())
            return Status::CANCELLED;
        auto it = messages.find(getKeyFromVmId(request->vmid()));
        response->set_value(!(it == messages.end()));
        cout << "HAS message " << request->value() << " ,vmid: " << request->vmid().processid() << "," << request->vmid().contractaddr() << endl;
        cout << boolalpha << "Result: " << response->value() << endl;
        return Status::OK;
    };
    Status Close(::grpc::ServerContext* context, const ::taraxa::vm::statedb::VmId* request, ::google::protobuf::Empty* response) {
        messages.erase(getKeyFromVmId(*request));
        return Status::OK;
    };

private:
    std::map<std::string, ::statedb::BytesMessage> messages;

    std::string getKeyFromVmId(const ::statedb::VmId vmId) {
        return vmId.processid() + "_" + vmId.contractaddr();
    }
};

void Start() {
    std::string server_address("0.0.0.0:50051");
    grpcServerImpl service;

    ServerBuilder builder;
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<Server> server(builder.BuildAndStart());
    std::cout << "Server listening on " << server_address << std::endl;
    std::cout << "Running..." << std::endl;
    server->Wait();
    std::cout << "Stop..." << std::endl;
}

// Example running server in separate thread
//auto server_ = Start();
//thread task(RunServer, server_);
//cout << boolalpha << task.get_id() << " joinable " << task.joinable() << endl;
//server_->Shutdown();
//task.join();

#endif //TESTS_INTEGRATION_GRPCSERVER_H
