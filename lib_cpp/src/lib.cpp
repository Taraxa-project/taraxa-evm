#include "taraxa_evm_cgo.h"
#include "taraxa_evm/cgo_bridge.hpp"
#include <iostream>

using namespace std;

namespace taraxa_evm::cgo_bridge {

    string run(const string &jsonConfig) {
        using namespace std;
        cout << jsonConfig << endl;
        auto cStr = const_cast<char *>(jsonConfig.c_str());
        string result = RunEvm(cStr);
        cout << result << endl;
        return result;
    }

}