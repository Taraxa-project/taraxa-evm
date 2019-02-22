cmake_minimum_required(VERSION 3.5)
project(tests_integration)

include(ExternalProject)

set(DEPS_DIR ${PROJECT_SOURCE_DIR}/deps)
set(GTEST_DIR ${DEPS_DIR}/googletest)
set(RAPIDJSON_DIR ${DEPS_DIR}/rapidjson)

set(CMAKE_CXX_STANDARD 14)

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

include_directories(
        ${PROJECT_SOURCE_DIR}/include
        ${GTEST_DIR}/include
        ${RAPIDJSON_DIR}/src/rapidjson/include
)
link_directories(${GTEST_DIR}/lib)

add_executable(tests_integration main.cpp)
add_dependencies(tests_integration googletest rapidjson)
target_link_libraries(tests_integration gtest_main gtest Threads::Threads)

enable_testing()
add_test(tests_integration "./tests_integration")

