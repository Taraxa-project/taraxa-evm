#ifndef TARAXA_EVM_UTIL_IO_HPP
#define TARAXA_EVM_UTIL_IO_HPP

#include <sstream>
#include <iostream>
#include <boost/process.hpp>
#include <boost/filesystem.hpp>

namespace taraxa_evm::__util_io {
    using namespace boost::process;
    using namespace boost::filesystem;
    using namespace std;

    string toString(ipstream &in) {
        stringstream ss;
        ss << in.rdbuf();
        return ss.str();
    }

    string createFreshTmpDir(const string &name) {
        auto tmpDirPath = temp_directory_path() / name;
        if (is_directory(tmpDirPath)) {
            remove_all(tmpDirPath);
        }
        create_directory(tmpDirPath);
        return tmpDirPath.string();
    }

}
namespace taraxa_evm::util_io {
    using __util_io::toString;
    using __util_io::createFreshTmpDir;
}
#endif //TARAXA_EVM_UTIL_IO_HPP
