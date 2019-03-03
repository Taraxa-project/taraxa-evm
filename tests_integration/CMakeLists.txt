cmake_minimum_required(VERSION 3.5)
project(tests_integration)

include(ExternalProject)

set(DEPS_DIR ${PROJECT_SOURCE_DIR}/deps)
set(GTEST_DIR ${DEPS_DIR}/googletest)
set(RAPIDJSON_DIR ${DEPS_DIR}/rapidjson)
# set protobuf files
set(PROTO_COMMON ../grpc/protobuf/common.proto)
set(PROTO_STATEDB ../grpc/protobuf/statedb.proto)

set(CMAKE_CXX_STANDARD 11)

# GRPC and Protocol Buffers libraries location
# change if needed by OS completition
# Debian example
#list(APPEND CMAKE_PREFIX_PATH "/usr/local")

# Cmake find modules
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_LIST_DIR}/cmake")

# threads and gtests for all tests
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

### This branch assumes that gRPC and all its dependencies are already installed
### on this system, so they can be located by find_package().

find_package(Protobuf REQUIRED)
find_package(GRPC REQUIRED)

set(PROTO_SRC_DIR ${CMAKE_CURRENT_BINARY_DIR}/proto-src)
file(MAKE_DIRECTORY ${PROTO_SRC_DIR})

### end grpc protobuf branch

# add headers
include_directories(
        ${PROJECT_SOURCE_DIR}/include
        ${GTEST_DIR}/include
        ${RAPIDJSON_DIR}/src/rapidjson/include
        ${PROTO_SRC_DIR}
)
link_directories(${GTEST_DIR}/lib)

protobuf_generate_cpp(PROTO_SRCS PROTO_HDRS ${PROTO_SRC_DIR} ${PROTO_COMMON})
grpc_generate_cpp(GRPC_SRCS GRPC_HDRS ${PROTO_SRC_DIR} ${PROTO_COMMON})
protobuf_generate_cpp(PROTO_SRCS PROTO_HDRS ${PROTO_SRC_DIR} ${PROTO_STATEDB})
grpc_generate_cpp(GRPC_SRCS GRPC_HDRS ${PROTO_SRC_DIR} ${PROTO_STATEDB})

# solidity contracts vm runnable testing
add_executable(tests_integration vm_tests.cpp)
add_dependencies(tests_integration googletest rapidjson)
target_link_libraries(tests_integration
        gtest_main
        gtest
        Threads::Threads
        )

# grpc/protobuf sync client/server testing
add_executable(tests_grpc grpc_tests.cpp
        include/grpcIntegrity.h
        include/grpcClient.h
        include/grpcServer.h
        ${PROTO_SRC_DIR}/statedb.pb.cc ${PROTO_SRC_DIR}/common.pb.cc ${PROTO_SRC_DIR}/statedb.grpc.pb.cc
        )

target_link_libraries(tests_grpc
        gRPC::grpc++_reflection
        protobuf::libprotobuf
        gtest_main
        gtest
        Threads::Threads
        )

add_executable(test_server server.cpp
        include/grpcServer.h
        ${PROTO_SRC_DIR}/statedb.pb.cc ${PROTO_SRC_DIR}/common.pb.cc ${PROTO_SRC_DIR}/statedb.grpc.pb.cc
        )

target_link_libraries(test_server
        gRPC::grpc++_reflection
        protobuf::libprotobuf
        gtest_main
        gtest
        Threads::Threads
        )

enable_testing()
add_test(tests_integration "./tests_integration")
add_test(tests_grpc "./tests_grpc")

