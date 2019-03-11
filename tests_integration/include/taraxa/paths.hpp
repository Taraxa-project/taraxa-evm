#ifndef TESTS_INTEGRATION_PATHS_HPP
#define TESTS_INTEGRATION_PATHS_HPP

#include <boost/filesystem.hpp>

namespace taraxa::paths {
    const auto INCLUDE_DIR = boost::filesystem::path(__FILE__)
            .parent_path()
            .parent_path();
    const auto PROJECT_DIR = INCLUDE_DIR.parent_path();
    const auto CONTRACTS_DIR = PROJECT_DIR / "contracts";
    const auto CONTRACTS_SRC_DIR = CONTRACTS_DIR / "src";
    const auto ROOT_DIR = PROJECT_DIR.parent_path();
    const auto EVM_EXECUTABLE = ROOT_DIR / "build" / "bin" / "evm";
}


#endif //TESTS_INTEGRATION_PATHS_HPP
