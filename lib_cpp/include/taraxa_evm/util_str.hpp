#ifndef TARAXA_EVM_UTIL_STRING_HPP
#define TARAXA_EVM_UTIL_STRING_HPP

#include <string>
#include <string.h>
#include <strings.h>
#include <cstring>
#include <tuple>
#include <iostream>
#include <sstream>
#include <boost/format.hpp>
#include <boost/process.hpp>
#include <boost/fusion/iterator.hpp>

namespace taraxa_evm::__util_str {
    using namespace std;
    using namespace boost;
    using namespace boost::fusion;

    template<class... Arg>
    string fmt(const string &pattern, const Arg &...arg) {
        stringstream ss;
        format formatter(pattern);
        for_each(std::make_tuple(arg...), [&](auto &e) {
            formatter = formatter % e;
        });
        ss << formatter;
        return ss.str();
    }

}
namespace taraxa_evm::util_str {
    using __util_str::fmt;
}

#endif //TARAXA_EVM_UTIL_STRING_HPP
