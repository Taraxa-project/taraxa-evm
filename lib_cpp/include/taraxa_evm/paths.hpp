#ifndef TARAXA_EVM_PATHS_HPP
#define TARAXA_EVM_PATHS_HPP

#include <boost/filesystem.hpp>

namespace taraxa_evm::paths {
    const auto INCLUDE_DIR = boost::filesystem::path(__FILE__)
            .parent_path()
            .parent_path();
    const auto PROJECT_DIR = INCLUDE_DIR.parent_path();
    const auto CONTRACTS_DIR = PROJECT_DIR / "contracts";
    const auto ROOT_DIR = PROJECT_DIR.parent_path();
}


#endif //TARAXA_EVM_PATHS_HPP
