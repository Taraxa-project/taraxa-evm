cmake_minimum_required(VERSION 3.5)
project(tests_integration)
set(CMAKE_CXX_STANDARD 17)

include(FetchContent)
include(ExternalProject)

set(SRC_DIR ${PROJECT_SOURCE_DIR}/src)
set(THIRDPARTY_DIR ${PROJECT_SOURCE_DIR}/thirdparty)
set(GTEST_DIR ${THIRDPARTY_DIR}/googletest)
set(RAPIDJSON_DIR ${THIRDPARTY_DIR}/rapidjson)
set(BOOST_DIR ${THIRDPARTY_DIR}/boost)
set(GRPC_DIR ${THIRDPARTY_DIR}/grpc)
set(GRPC_GEN_DIR ${PROJECT_SOURCE_DIR}/../grpc/grpc_cpp)

set(FETCHCONTENT_BASE_DIR ${THIRDPARTY_DIR})

find_package(Threads REQUIRED)

FetchContent_Declare(
        googletest_project
        GIT_REPOSITORY https://github.com/google/googletest
        GIT_TAG v1.8.1
)
FetchContent_Declare(
        rapidjson_project
        GIT_REPOSITORY https://github.com/Tencent/rapidjson
        GIT_TAG v1.1.0
)
FetchContent_Declare(
        boost_project
        GIT_REPOSITORY https://github.com/boostorg/boost
        GIT_TAG boost-1.69.0
)
FetchContent_Declare(
        grpc_project
        GIT_REPOSITORY https://github.com/grpc/grpc
        GIT_TAG v1.19.0
)

FetchContent_MakeAvailable(googletest_project rapidjson_project boost_project grpc_project)

#ExternalProject_Add(
#        googletest_project
#        PREFIX ${GTEST_DIR}
#        GIT_REPOSITORY https://github.com/google/googletest
#        GIT_TAG v1.8.1
#        TEST_COMMAND ""
#)

#ExternalProject_Add(
#        rapidjson_project
#        PREFIX ${RAPIDJSON_DIR}
#        GIT_REPOSITORY https://github.com/Tencent/rapidjson
#        GIT_TAG v1.1.0
#        INSTALL_COMMAND ""
#        CONFIGURE_COMMAND ""
#        BUILD_COMMAND ""
#        TEST_COMMAND ""
#)

#ExternalProject_Add(
#        boost_project
#        PREFIX ${BOOST_DIR}
#        GIT_REPOSITORY https://github.com/boostorg/boost
#        GIT_TAG boost-1.69.0
#        #        GIT_SUBMODULES
#        #            libs/config libs/core libs/iterator libs/io libs/functional libs/mpl
#        #            libs/preprocessor libs/static_assert libs/system libs/detail
#        #            libs/type_traits libs/utility libs/assert libs/filesystem
#        INSTALL_COMMAND ""
#        CONFIGURE_COMMAND ""
#        BUILD_COMMAND ""
#        TEST_COMMAND ""
#)

#ExternalProject_Add(
#        grpc_project
#        PREFIX ${GRPC_DIR}
#        GIT_REPOSITORY https://github.com/grpc/grpc
#        GIT_TAG v1.19.0
#        INSTALL_COMMAND ""
#        CONFIGURE_COMMAND ""
#        BUILD_COMMAND ""
#        TEST_COMMAND ""
#        #        CMAKE_CACHE_ARGS
#        #        -Dprotobuf_BUILD_TESTS:BOOL=OFF
#        #        -Dprotobuf_WITH_ZLIB:BOOL=OFF
#        #        -Dprotobuf_MSVC_STATIC_RUNTIME:BOOL=OFF
#        #        -DCMAKE_INSTALL_PREFIX:PATH=${CMAKE_CURRENT_BINARY_DIR}/protobuf
#)


#add_subdirectory(${PROJECT_SOURCE_DIR}/../thirdparty/grpc)

#find_package(Protobuf REQUIRED)
#find_package(GRPC REQUIRED)

FILE(GLOB BOOST_INCLUDE_PATHS ${boost_project_SOURCE_DIR}/libs/**/include)
FILE(GLOB BOOST_FILESYSTEM_SRC ${boost_project_DIR}/filesystem/src/*)
FILE(GLOB GRPC_GEN_FILES ${GRPC_GEN_DIR}/*)

include_directories(
        ${BOOST_INCLUDE_PATHS}
        ${rapidjson_project_SOURCE_DIR}/include
        ${googletest_project_SOURCE_DIR}/include
        ${GRPC_GEN_DIR}
        ${PROJECT_SOURCE_DIR}
)

add_executable(
        ${PROJECT_NAME}
        ${BOOST_FILESYSTEM_SRC}
        ${GRPC_GEN_FILES}
        ${SRC_DIR}/main.cpp
)

target_link_libraries(
        ${PROJECT_NAME}
        gtest_main
        gtest
        grpc++_reflection
        libprotobuf
        Threads::Threads
)

#add_dependencies(${PROJECT_NAME} googletest_project rapidjson_project boost_project grpc_project)

enable_testing()
add_test(tests_integration "./tests_integration")
add_test(tests_grpc "./tests_grpc")

