#ifndef TARAXAGTESTS_PROCESSTEST_H
#define TARAXAGTESTS_PROCESSTEST_H

#include <string>
#include "Process.h"

using namespace std;

string TestBlindAuction() {
    // OK
    const char* cmd = "../evm --codefile ../clear_contracts/BlindAuction.bin --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestAcknowledger() {
    // OK
    const char* cmd = "../evm --codefile ../clear_contracts/Acknowledger.bin --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestCodeVerify() {
    // verify code
    const char* cmd = "../evm --codefile ../crash_contracts/unverify.bin --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestVulnerabilityRecursive() {
    // crash vulnerability
    const char* cmd = "../evm --codefile ../crash_contracts/recursive.bin --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestVulnerabilityLoop() {
    // crash vulnerability
    const char* cmd = "../evm --codefile ../crash_contracts/loop.bin --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestGasOK() {
    // OK gas
    const char* cmd = "../evm --code 6040 --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

string TestOutOfGas() {
    // out of gas
    const char* cmd = "../evm --code 6040 --gas 0x1 --price 0x3 --json run";

    Document doc;
    evmJsonOutput output;
    output.Exec(cmd);
    cout << "Result: " << output.GetRegexResult() << endl;

    doc.Parse(output.GetRegexResult().c_str());

    evmJsonOutput result = output.fromJSON(doc);

    cout << "Output: " << result.getOutput() << endl;
    cout << "gasUsed: " << result.getGasUsed() << endl;
    cout << "Time Execution, ns: " << result.getTime() << endl;
    cout << "Error message: " << result.getError() << endl;

    return result.getError();
}

#endif //TARAXAGTESTS_PROCESSTEST_H
