#include "taraxa_evm_cgo.h"
#include "taraxa_evm/cgo_bridge.hpp"

using namespace std;

namespace taraxa_evm::cgo_bridge {

    string run(const string &jsonConfig) {
        auto cStr = const_cast<char *>(jsonConfig.c_str());
        return RunEvm(cStr);
    }

}