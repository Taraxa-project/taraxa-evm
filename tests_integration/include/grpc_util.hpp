#ifndef TESTS_INTEGRATION_GRPC_UTIL_HPP
#define TESTS_INTEGRATION_GRPC_UTIL_HPP

#include <iostream>
#include <grpc/grpc.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

using namespace std;

unique_ptr<Server> startGRPCService(grpc::Service *service, const string &server_address) {
    grpc::ServerBuilder serverBuilder;
    serverBuilder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    serverBuilder.RegisterService(service);
    cout << "Starting server on " << server_address << endl;
    return serverBuilder.BuildAndStart();
}

#endif //TESTS_INTEGRATION_GRPC_UTIL_HPP
