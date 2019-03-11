#ifndef TESTS_INTEGRATION_GRPC_UTIL_HPP
#define TESTS_INTEGRATION_GRPC_UTIL_HPP

#include <iostream>
#include <grpc/grpc.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

namespace taraxa::__util_grpc {
    using namespace std;
    using namespace grpc;

    unique_ptr<Server> startGRPCService(Service *service, const string &server_address) {
        ServerBuilder serverBuilder;
        serverBuilder.AddListeningPort(server_address, InsecureServerCredentials());
        serverBuilder.RegisterService(service);
        cout << "Starting server on " << server_address << endl;
        return serverBuilder.BuildAndStart();
    }

}
namespace taraxa::util_grpc {
    using __util_grpc::startGRPCService;
}

#endif //TESTS_INTEGRATION_GRPC_UTIL_HPP
