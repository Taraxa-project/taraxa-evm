extern "C" {
#include "taraxa_evm_cgo.h"
}

#include "taraxa_evm/cgo_bridge.hpp"

using namespace std;

namespace taraxa_evm {

    string runEvm(const string &jsonConfig, const ExternalApi &externalApi) {
        return RunEvm(
                const_cast<char *>(jsonConfig.c_str()),
                const_cast<ExternalApi *>(&externalApi)
        );
    }

}