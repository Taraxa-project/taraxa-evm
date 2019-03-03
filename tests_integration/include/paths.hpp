#ifndef TESTS_INTEGRATION_PATHS_HPP
#define TESTS_INTEGRATION_PATHS_HPP

#include <boost/filesystem.hpp>


namespace paths {
    namespace fs = boost::filesystem;

    const auto INCLUDE_DIR = fs::path(__FILE__).parent_path();
    const auto PROJECT_DIR = INCLUDE_DIR.parent_path();
    const auto RESOURCES_DIR = PROJECT_DIR / "resources";
    const auto ROOT_DIR = PROJECT_DIR.parent_path();
    const auto EVM_EXECUTABLE = ROOT_DIR / "build" / "bin" / "evm";
}


#endif //TESTS_INTEGRATION_PATHS_HPP
