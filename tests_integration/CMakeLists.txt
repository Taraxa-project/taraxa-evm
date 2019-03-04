cmake_minimum_required(VERSION 3.13)
project(tests_integration)
set(CMAKE_CXX_STANDARD 17)

include(FetchContent)

set(SRC_DIR ${PROJECT_SOURCE_DIR}/src)
set(GRPC_GEN_DIR ${PROJECT_SOURCE_DIR}/../grpc/grpc_cpp)

set(FETCHCONTENT_BASE_DIR ${PROJECT_SOURCE_DIR}/thirdparty)
set(FETCHCONTENT_QUIET OFF)

function(git_dependency name repo tag)
    FetchContent_Declare(
            ${name}
            GIT_REPOSITORY ${repo}
            GIT_TAG ${tag}
            GIT_SHALLOW ON
            GIT_PROGRESS ON
            GIT_SUBMODULES ${ARGN}
    )
endfunction()

git_dependency(googletest_project https://github.com/google/googletest release-1.8.0 .)
git_dependency(rapidjson_project https://github.com/Tencent/rapidjson v1.1.0 .)
git_dependency(grpc_project https://github.com/grpc/grpc v1.19.0
        third_party/protobuf third_party/benchmark third_party/gflags
        third_party/zlib third_party/cares third_party/boringssl)
# TODO latest
git_dependency(boost_project https://github.com/boostorg/boost boost-1.66.0
        libs/config libs/core libs/iterator libs/io libs/functional libs/function libs/function_types
        libs/mpl libs/predef libs/preprocessor libs/static_assert libs/system libs/detail libs/throw_exception
        libs/exception libs/smart_ptr libs/range libs/type_traits libs/utility libs/assert libs/filesystem)

foreach (p googletest_project rapidjson_project boost_project grpc_project)
    FetchContent_Populate(${p})
endforeach (p)

foreach (p googletest_project grpc_project)
    add_subdirectory(${${p}_SOURCE_DIR} EXCLUDE_FROM_ALL)
endforeach (p)

find_package(Threads REQUIRED)

FILE(GLOB BOOST_INCLUDE_PATHS ${boost_project_SOURCE_DIR}/libs/**/include)
FILE(GLOB BOOST_FILESYSTEM_SRC ${boost_project_SOURCE_DIR}/libs/filesystem/src/*)
FILE(GLOB BOOST_SYSTEM_SRC ${boost_project_SOURCE_DIR}/libs/system/src/*)
FILE(GLOB GRPC_GEN_FILES ${GRPC_GEN_DIR}/*)

include_directories(
        ${BOOST_INCLUDE_PATHS}
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
)

enable_testing()
add_test(${PROJECT_NAME} "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}")
