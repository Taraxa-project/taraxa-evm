cmake_minimum_required(VERSION 3.5)
project(tests_integration)

include(ExternalProject)

set(DEPS_DIR ${PROJECT_SOURCE_DIR}/deps)
set(GTEST_DIR ${DEPS_DIR}/googletest)
set(RAPIDJSON_DIR ${DEPS_DIR}/rapidjson)
set(PROTO_DIR ../grpc/grpc_cpp)

set(CMAKE_CXX_STANDARD 14)

list(APPEND CMAKE_PREFIX_PATH "/usr/local")

# Cmake find modules
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_LIST_DIR}/cmake")

find_package(Threads REQUIRED)
ExternalProject_Add(googletest
    GIT_REPOSITORY https://github.com/google/googletest
    CMAKE_ARGS -DCMAKE_INSTALL_PREFIX=${GTEST_DIR}
    PREFIX ${GTEST_DIR}
)
ExternalProject_Add(rapidjson
    GIT_REPOSITORY https://github.com/Tencent/rapidjson
    INSTALL_COMMAND ""
    CONFIGURE_COMMAND ""
    BUILD_COMMAND ""
    TEST_COMMAND ""
    PREFIX ${RAPIDJSON_DIR}
)

# This branch assumes that gRPC and all its dependencies are already installed
# on this system, so they can be located by find_package().

# Find Protobuf installation
# Looks for protobuf-config.cmake file installed by Protobuf's cmake installation.
set(protobuf_MODULE_COMPATIBLE TRUE)
find_package(Protobuf REQUIRED)
message(STATUS "Using protobuf ${protobuf_VERSION}")

set(_PROTOBUF_LIBPROTOBUF protobuf::libprotobuf)
set(_PROTOBUF_PROTOC $<TARGET_FILE:protobuf::protoc>)

# Find gRPC installation
# Looks for gRPCConfig.cmake file installed by gRPC's cmake installation.
find_package(GRPC REQUIRED)
message(STATUS "Using gRPC ${gRPC_VERSION}")

set(_GRPC_GRPCPP_UNSECURE GRPC::grpc++_unsecure)
set(_GRPC_CPP_PLUGIN_EXECUTABLE $<TARGET_FILE:GRPC::grpc_cpp_plugin>)


include_directories(
        ${PROJECT_SOURCE_DIR}/include
        ${GTEST_DIR}/include
        ${RAPIDJSON_DIR}/src/rapidjson/include
        ${PROTO_DIR}
)
link_directories(${GTEST_DIR}/lib)

add_executable(tests_integration main.cpp ${PROTO_DIR}/statedb.pb.cc ${PROTO_DIR}/common.pb.cc ${PROTO_DIR}/statedb.grpc.pb.cc include/grpcServer.h)
add_dependencies(tests_integration googletest rapidjson)
target_link_libraries(tests_integration
        gtest_main
        gtest
        Threads::Threads
        gRPC::grpc++_reflection
        protobuf::libprotobuf
        )

enable_testing()
add_test(tests_integration "./tests_integration")

