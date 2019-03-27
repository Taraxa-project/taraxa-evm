cmake_minimum_required(VERSION 3.13)
project(taraxa_evm)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_MACOSX_RPATH 1)

include(FetchContent)

set(SRC_DIR ${PROJECT_SOURCE_DIR}/src)
set(ROOT_DIR ${PROJECT_SOURCE_DIR}/..)
set(GO_MAIN_PACKAGE ${ROOT_DIR}/main)
set(CGO_LIB_DEST ${CMAKE_CURRENT_BINARY_DIR}/taraxa_evm_cgo)
set(CGO_LIB_FILE ${CGO_LIB_DEST}/taraxa_evm_cgo.a)

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

git_dependency(boost_cmake https://github.com/Orphis/boost-cmake v1.67.0)
git_dependency(googletest_project https://github.com/google/googletest release-1.8.0 .)
git_dependency(rapidjson_project https://github.com/Tencent/rapidjson v1.1.0 .)

foreach (p boost_cmake googletest_project)
    add_subdirectory(${${p}_SOURCE_DIR} EXCLUDE_FROM_ALL)
endforeach (p)

#find_package(Threads REQUIRED)

include_directories(
        ${BOOST_SOURCE}
        ${rapidjson_project_SOURCE_DIR}/include
        ${googletest_project_SOURCE_DIR}/googletest/include
        ${googletest_project_SOURCE_DIR}/googlemock/include
        ${CGO_LIB_DEST}
        ${PROJECT_SOURCE_DIR}/include
)

add_custom_target(
        taraxa_vm_cgo_shared_lib_build

        COMMAND go build -tags=lib_cpp -buildmode=c-archive -o ${CGO_LIB_FILE}
        WORKING_DIRECTORY ${GO_MAIN_PACKAGE}
        COMMENT "Build CGO static lib from ${GO_MAIN_PACKAGE} to ${CGO_LIB_DEST}"
)
add_library(taraxa_vm_cgo_shared_lib STATIC IMPORTED)
set_target_properties(taraxa_vm_cgo_shared_lib PROPERTIES IMPORTED_LOCATION ${CGO_LIB_FILE})
add_dependencies(taraxa_vm_cgo_shared_lib taraxa_vm_cgo_shared_lib_build)

add_library(${PROJECT_NAME} STATIC ${SRC_DIR}/lib.cpp)
target_link_libraries(
        ${PROJECT_NAME}
        taraxa_vm_cgo_shared_lib
)
if (APPLE)
    target_link_libraries(
            ${PROJECT_NAME}
            "-framework CoreFoundation"
            "-framework Security"
    )
endif (APPLE)

add_executable(
        ${PROJECT_NAME}_tests
        ${SRC_DIR}/tests.cpp
)
target_link_libraries(
        ${PROJECT_NAME}_tests
        ${PROJECT_NAME}
        gtest_main
        gtest
        #        Threads::Threads
        Boost::filesystem
)

enable_testing()
add_test(${PROJECT_NAME} "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}_tests")
