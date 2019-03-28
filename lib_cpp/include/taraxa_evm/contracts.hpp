#ifndef TARAXA_EVM_CONTRACTS_HPP
#define TARAXA_EVM_CONTRACTS_HPP

#include <string>
#include <iostream>
#include <boost/process.hpp>
#include "paths.hpp"
#include "util_io.hpp"

// TODO not used for now
namespace taraxa::__contracts {
    using namespace taraxa::util_io;
    using namespace boost::process;
    using namespace std;
    using namespace paths;

    template<class... Arg>
    string callContractCli(const string &cmd, const string &contractName, const Arg &...args) {
        ipstream stdOut;
        child nodeProcess(search_path("node"), CONTRACTS_DIR.string(),
                          cmd, contractName + ".sol", contractName, args...,
                          std_out > stdOut);
        nodeProcess.join();
        return toString(stdOut);
    }

    template<class... Arg>
    string generateCall(const string &contractName, const Arg &...args) {
        return callContractCli("generate_call", contractName, args...);
    }

    string getCode(const string &contractName) {
        return callContractCli("get_code", contractName);
    }

}
namespace taraxa::contracts {
    using __contracts::generateCall;
    using __contracts::getCode;
}

#endif //TARAXA_EVM_CONTRACTS_HPP
