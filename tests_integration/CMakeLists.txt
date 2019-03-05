cmake_minimum_required(VERSION 3.13)
project(tests_integration)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_MACOSX_RPATH 1)

include(FetchContent)

set(SRC_DIR ${PROJECT_SOURCE_DIR}/src)
set(GRPC_GEN_DIR ${PROJECT_SOURCE_DIR}/../grpc/grpc_cpp)

set(FETCHCONTENT_BASE_DIR ${PROJECT_SOURCE_DIR}/thirdparty)
set(FETCHCONTENT_QUIET OFF)

macro(git_dependency name repo tag)
    FetchContent_Declare(
            ${name}
            GIT_REPOSITORY ${repo}
            GIT_TAG ${tag}
            GIT_SHALLOW ON
            GIT_PROGRESS ON
            GIT_SUBMODULES ${ARGN}
    )
    FetchContent_Populate(${name})
endmacro()

# TODO latest version
git_dependency(boost_cmake https://github.com/Orphis/boost-cmake v1.67.0)
add_subdirectory(${boost_cmake_SOURCE_DIR})
#set(BOOST_ROOT ${BOOST_SOURCE}) TODO
#foreach (lib regex filesystem unit_test_framework program_options system)
#    string(TOUPPER ${lib} lib_upper)
#    set(Boost_${lib_upper}_FOUND ON)
#endforeach (lib)

git_dependency(googletest_project https://github.com/google/googletest release-1.8.0 .)
git_dependency(rapidjson_project https://github.com/Tencent/rapidjson v1.1.0 .)
git_dependency(grpc_project https://github.com/grpc/grpc v1.19.0
        third_party/protobuf third_party/benchmark third_party/gflags
        third_party/zlib third_party/cares third_party/boringssl)

# TODO git_dependency(solidity_project https://github.com/ethereum/solidity v0.5.4 .)
foreach (p googletest_project grpc_project)
    add_subdirectory(${${p}_SOURCE_DIR} EXCLUDE_FROM_ALL)
endforeach (p)

find_package(Threads REQUIRED)

file(GLOB BOOST_FILESYSTEM_SRC ${boost_project_SOURCE_DIR}/libs/filesystem/src/*)
file(GLOB BOOST_SYSTEM_SRC ${boost_project_SOURCE_DIR}/libs/system/src/*)
file(GLOB GRPC_GEN_FILES ${GRPC_GEN_DIR}/*)

include_directories(
        ${BOOST_SOURCE}
        ${rapidjson_project_SOURCE_DIR}/include
        ${googletest_project_SOURCE_DIR}/googletest/include
        ${googletest_project_SOURCE_DIR}/googlemock/include
        ${GRPC_GEN_DIR}
        ${PROJECT_SOURCE_DIR}
)

add_executable(
        ${PROJECT_NAME}
        ${BOOST_FILESYSTEM_SRC}
        ${BOOST_SYSTEM_SRC}
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
        Boost::filesystem
)

enable_testing()
add_test(${PROJECT_NAME} "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}")
