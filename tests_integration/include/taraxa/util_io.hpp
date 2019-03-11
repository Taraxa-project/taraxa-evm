#ifndef TESTS_INTEGRATION_IO_UTIL_HPP
#define TESTS_INTEGRATION_IO_UTIL_HPP

#include <sstream>
#include <iostream>
#include <boost/process.hpp>

namespace taraxa::__util_io {
    using namespace boost::process;
    using namespace std;

    string toString(ipstream &in) {
        stringstream ss;
        ss << in.rdbuf();
        return ss.str();
    }

}
namespace taraxa::util_io {
    using __util_io::toString;
}
#endif //TESTS_INTEGRATION_IO_UTIL_HPP
