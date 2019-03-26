#ifndef TARAXA_EVM_UTIL_IO_HPP
#define TARAXA_EVM_UTIL_IO_HPP

#include <sstream>
#include <iostream>
#include <boost/process.hpp>

namespace taraxa_evm::__util_io {
    using namespace boost::process;
    using namespace std;

    string toString(ipstream &in) {
        stringstream ss;
        ss << in.rdbuf();
        return ss.str();
    }

}
namespace taraxa_evm::util_io {
    using __util_io::toString;
}
#endif //TARAXA_EVM_UTIL_IO_HPP
