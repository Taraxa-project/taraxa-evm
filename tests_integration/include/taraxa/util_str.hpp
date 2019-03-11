//
// Created by John Doe on 2019-03-10.
//

#ifndef TESTS_INTEGRATION_STRING_UTIL_HPP
#define TESTS_INTEGRATION_STRING_UTIL_HPP

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

namespace taraxa::__util_str {
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
namespace taraxa::util_str {
    using __util_str::fmt;
}

#endif //TESTS_INTEGRATION_STRING_UTIL_HPP
