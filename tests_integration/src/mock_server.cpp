#include "taraxa/StateDBMockServer.hpp"
#include "taraxa/util_grpc.hpp"

int main(int argc, char **argv) {
    using namespace taraxa;
    util_grpc::startGRPCService(new StateDBMockServer(), argv[1])->Wait();
    return 0;
}