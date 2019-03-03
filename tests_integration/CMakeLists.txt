cmake_minimum_required(VERSION 3.5)
project(tests_integration)
set(CMAKE_CXX_STANDARD 17)

include(ExternalProject)

set(SRC_DIR ${PROJECT_SOURCE_DIR}/src)
set(THIRDPARTY_DIR ${PROJECT_SOURCE_DIR}/thirdparty)
set(GTEST_DIR ${THIRDPARTY_DIR}/googletest)
set(RAPIDJSON_DIR ${THIRDPARTY_DIR}/rapidjson)
#set(QTBASE_DIR ${THIRDPARTY_DIR}/qtbase)
set(BOOST_DIR ${THIRDPARTY_DIR}/boost)
set(PROTOBUF_DIR ${THIRDPARTY_DIR}/protobuf)
set(GRPC_DIR ${THIRDPARTY_DIR}/grpc)
set(GRPC_GEN_DIR ${PROJECT_SOURCE_DIR}/../grpc/grpc_cpp)


find_package(Threads REQUIRED)
ExternalProject_Add(
        googletest
        PREFIX ${GTEST_DIR}
        GIT_REPOSITORY https://github.com/google/googletest
        TEST_COMMAND ""
)
ExternalProject_Add(
        rapidjson
        PREFIX ${RAPIDJSON_DIR}
        GIT_REPOSITORY https://github.com/Tencent/rapidjson
        INSTALL_COMMAND ""
        CONFIGURE_COMMAND ""
        BUILD_COMMAND ""
        TEST_COMMAND ""
)

#ExternalProject_Add(
#        qtbase
#        PREFIX ${QTBASE_DIR}
#        GIT_REPOSITORY https://github.com/qt/qtbase
#        GIT_TAG v5.12.1
#        INSTALL_COMMAND ""
#        CONFIGURE_COMMAND ""
#        BUILD_COMMAND ""
#        TEST_COMMAND ""
#)

ExternalProject_Add(
        boost
        PREFIX ${BOOST_DIR}
        GIT_REPOSITORY https://github.com/boostorg/boost
        GIT_TAG boost-1.69.0
        #        GIT_SUBMODULES
        #            libs/config libs/core libs/iterator libs/io libs/functional libs/mpl
        #            libs/preprocessor libs/static_assert libs/system libs/detail
        #            libs/type_traits libs/utility libs/assert libs/filesystem
        INSTALL_COMMAND ""
        CONFIGURE_COMMAND ""
        BUILD_COMMAND ""
        TEST_COMMAND ""
)

#ExternalProject_Add(
#        protobuf
#        PREFIX ${PROTOBUF_DIR}
#        GIT_REPOSITORY https://github.com/protocolbuffers/protobuf.git
#        CONFIGURE_COMMAND ${CMAKE_COMMAND} ${PROTOBUF_DIR}/src/protobuf/cmake
#        TEST_COMMAND ""
#        #        CMAKE_CACHE_ARGS
#        #        -Dprotobuf_BUILD_TESTS:BOOL=OFF
#        #        -Dprotobuf_WITH_ZLIB:BOOL=OFF
#        #        -Dprotobuf_MSVC_STATIC_RUNTIME:BOOL=OFF
#        #        -DCMAKE_INSTALL_PREFIX:PATH=${CMAKE_CURRENT_BINARY_DIR}/protobuf
#)

ExternalProject_Add(
        grpc_project
        PREFIX ${GRPC_DIR}
        GIT_REPOSITORY https://github.com/grpc/grpc
        INSTALL_COMMAND ""
        CONFIGURE_COMMAND ""
        BUILD_COMMAND ""
        TEST_COMMAND ""
        #        CMAKE_CACHE_ARGS
        #        -Dprotobuf_BUILD_TESTS:BOOL=OFF
        #        -Dprotobuf_WITH_ZLIB:BOOL=OFF
        #        -Dprotobuf_MSVC_STATIC_RUNTIME:BOOL=OFF
        #        -DCMAKE_INSTALL_PREFIX:PATH=${CMAKE_CURRENT_BINARY_DIR}/protobuf
)

add_subdirectory(${GRPC_DIR}/src/grpc)

#find_package(Protobuf REQUIRED)
#find_package(GRPC REQUIRED)

FILE(GLOB BOOST_INCLUDE_PATHS ${BOOST_DIR}/src/boost/libs/**/include)
FILE(GLOB BOOST_FILESYSTEM_SRC ${BOOST_DIR}/src/boost/libs/filesystem/src/*)
FILE(GLOB GRPC_GEN_FILES ${GRPC_GEN_DIR}/*)
MESSAGE(${BOOST_INCLUDE_PATHS})

include_directories(
        #        ${QTBASE_DIR}/src/qtbase/src/corelib/io
        ${BOOST_INCLUDE_PATHS}
        ${RAPIDJSON_DIR}/src/rapidjson/include
        ${GTEST_DIR}/include
        ${GRPC_GEN_DIR}
        ${PROJECT_SOURCE_DIR}
)

add_executable(tests_integration ${BOOST_FILESYSTEM_SRC} ${SRC_DIR}/vm_tests.cpp)

target_link_libraries(
        tests_integration
        gtest_main
        gtest
        Threads::Threads
)

add_executable(
        tests_grpc
        ${SRC_DIR}/grpc_tests.cpp
        ${GRPC_GEN_FILES}
)

target_link_libraries(
        tests_grpc
        grpc++_reflection
        libprotobuf
        gtest_main
        gtest
        Threads::Threads
)

add_executable(
        test_server
        ${SRC_DIR}/server.cpp
        ${GRPC_GEN_FILES}
)

target_link_libraries(
        test_server
        grpc++_reflection
        libprotobuf
        gtest_main
        gtest
        Threads::Threads
)

add_dependencies(tests_integration googletest rapidjson boost grpc)
add_dependencies(tests_grpc googletest rapidjson boost grpc)
add_dependencies(test_server googletest rapidjson boost grpc)

enable_testing()
add_test(tests_integration "./tests_integration")
add_test(tests_grpc "./tests_grpc")

