#ifndef TARAXAGTESTS_PROCESS_H
#define TARAXAGTESTS_PROCESS_H

#include <cstdio>
#include <iostream>
#include <memory>
#include <stdexcept>
#include <string>
#include <array>
#include <iostream>
#include <rapidjson/document.h>
#include <boost/filesystem.hpp>
#include "paths.hpp"

using namespace std;
using namespace rapidjson;
using namespace boost::filesystem;

class Process {

public:

    Process() {};

    ~Process() {};

    void Exec(const char *cmd) {
        array<char, 128> buffer{};
        result.clear();
        unique_ptr<FILE, decltype(&pclose)> pipe(popen(cmd, "r"), pclose);
        if (!pipe) {
            throw runtime_error("popen()|pclose() failed!");
        }

        while (fgets(buffer.data(), buffer.size(), pipe.get()) != nullptr) {
            result += buffer.data();
        }

        if (result.size() <= 2 && result[0] != '{' && result.substr(0, 2) != "0x" && result[0] != '[') {
            throw runtime_error(result.c_str());
        }
    };

    string GetResult() { return result; };

    string GetRegexResult() {
        stringstream ss(result.c_str());
        string to;

        while (getline(ss, to, '\n')) {
            if (to.substr(0, pattern.size()) == pattern)
                return to;
        }
        return "";
    };

protected:

    string result;
    string pattern;

};

class evmJsonOutput : public Process {

public:

    evmJsonOutput() {
        this->pattern = "{\"output\"";
    };

    evmJsonOutput(string output, string gas_used, unsigned time, string error)
            : output(output),
              gasUsed(gas_used),
              time(time),
              error(error) {
        this->pattern = "{\"output\"";
    };

    ~evmJsonOutput() {};

    void setOutput(string output) {
        this->output = output;
    }

    void setGasUsed(string gasUsed) {
        this->gasUsed = gasUsed;
    }

    void setTime(unsigned time) {
        this->time = time;
    }

    void setError(string error) {
        this->error = error;
    }

    string getOutput() {
        return this->output;
    }

    string getGasUsed() {
        return this->gasUsed;
    }

    unsigned getTime() {
        return this->time;
    }

    string getError() {
        return this->error;
    }

    static evmJsonOutput fromJSON(Document &doc) {
        if (!doc.IsObject())
            throw runtime_error("document should be an object");

        static const char *members[] = {"output", "gasUsed", "time", "error"};
        string _output, _gas_used, _error;
        unsigned _time;

        if (doc.HasMember(members[0]))
            _output = doc[members[0]].GetString();
        if (doc.HasMember(members[1]))
            _gas_used = doc[members[1]].GetString();
        if (doc.HasMember(members[2]))
            _time = doc[members[2]].GetUint();
        if (doc.HasMember(members[3]))
            _error = doc[members[3]].GetString();

        evmJsonOutput result(_output, _gas_used, _time, _error);
        return result;
    }

private:

    string output;
    string gasUsed;
    unsigned time;
    string error;

};


static string RunTest(const string &args) {
    string cmd = paths::EVM_EXECUTABLE.string() + " --verbosity 3 " + args;
    Document doc;
    evmJsonOutput output;
    output.Exec(cmd.c_str());
    cout << "Result: " << output.GetRegexResult() << endl;
    doc.Parse(output.GetRegexResult().c_str());
    evmJsonOutput result = output.fromJSON(doc);
    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;
    return result.getError();

}

static string RunCodeFile(const string &codeFile, const string &args = "") {
    return RunTest("--codefile " + (paths::RESOURCES_DIR / codeFile).string() + " --json " + args + " run");
}

#endif //TARAXAGTESTS_PROCESS_H
