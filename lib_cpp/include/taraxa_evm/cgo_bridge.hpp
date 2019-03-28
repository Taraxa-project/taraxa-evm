#ifndef TARAXA_EVM_CGO_BRIDGE_HPP
#define TARAXA_EVM_CGO_BRIDGE_HPP

#include <string>

extern "C" {

#include "cgo_imports.h"

}

namespace taraxa_evm {

    using ExternalApi = ExternalApi;

    std::string runEvm(const std::string &, const ExternalApi &externalApi);

}

#endif //TARAXA_EVM_CGO_BRIDGE_HPP
