cmake_minimum_required(VERSION 3.5)
project(TaraxaGTests)

find_package(GTest REQUIRED)
find_package(Threads REQUIRED)

set(CMAKE_CXX_STANDARD 14)

include_directories(
        ../include
        ${GTEST_INCLUDE_DIRS}
)

add_executable(TaraxaGTests main.cpp include/Process.h include/ProcessTest.h)


target_link_libraries(
        TaraxaGTests ${GTEST_LIBRARIES} Threads::Threads)

enable_testing()
add_test(TaraxaGTests "./TaraxaGTests")

