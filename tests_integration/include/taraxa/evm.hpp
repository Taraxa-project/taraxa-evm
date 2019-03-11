#ifndef TESTS_INTEGRATION_EVM_HPP
#define TESTS_INTEGRATION_EVM_HPP

#include <stdexcept>
#include <boost/format.hpp>
#include "paths.hpp"
#include "util_io.hpp"
#include "util_str.hpp"
#include "util_json.hpp"
#include "contracts.hpp"
#include <rapidjson/document.h>

namespace taraxa::__evm {
    using namespace std;
    using namespace boost;
    using namespace boost::process;
    using namespace rapidjson;
    using namespace taraxa::paths;
    using namespace taraxa::util_io;
    using namespace taraxa::util_str;
    using namespace taraxa::util_json;
    using namespace taraxa::contracts;

    struct JsonOutput {

        string output;
        string gasUsed;
        int time;
        string error;

        static JsonOutput from(const string &str) {
            Document doc;
            doc.Parse(str.c_str());
            return JsonOutput{
                    .output = get(doc, "output", ""),
                    .gasUsed = get(doc, "gasUsed", ""),
                    .time = get(doc, "time", 0),
                    .error = get(doc, "error", "")
            };
        }

    };

    template<class... Arg>
    string cli(const Arg &...args) {
        ipstream stdOut, stdErr;
        child process(EVM_EXECUTABLE, args..., std_out > stdOut, std_err > stdErr);
        auto pid = process.id();
        cout << fmt("Launched evm with pid %s and args: ", pid) + ((string(" ") + args) + ...) + "\n";
        process.join();
        int code = process.exit_code();
        auto out = toString(stdOut);
        if (code != 0) {
            throw runtime_error(fmt(
                    "EVM with pid %s exited with code %s.\nStdout:\n%s\nStderr:\n%s", pid, code, out, toString(stdErr)
            ));
        }
        cout << fmt("Evm with pid %s has exited normally with stdout:\n%s\n", pid, out);
        return out;
    }

    template<class... Arg>
    JsonOutput run(const Arg &...args) {
        auto out = cli("--verbosity", "3", "--json", args..., "run");
        auto lastLineStart = out.rfind('\n', out.length() - 2);
        auto lastLine = out.substr(lastLineStart == string::npos ? 0 : lastLineStart, string::npos);
        return JsonOutput::from(lastLine);
    }

    template<class... Arg>
    JsonOutput runCode(const string &code, const Arg &...args) {
        return run("--code", code, args...);
    }

    template<class... Arg>
    JsonOutput runFile(const string &codeFile, const Arg &...args) {
        return run("--codefile", (CONTRACTS_SRC_DIR / codeFile).string(), args...);
    }

}
namespace taraxa::evm {
    using __evm::cli;
    using __evm::run;
    using __evm::runCode;
    using __evm::runFile;
    using __evm::JsonOutput;
}

#endif //TESTS_INTEGRATION_EVM_HPP