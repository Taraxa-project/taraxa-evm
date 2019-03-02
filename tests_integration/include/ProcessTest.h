#ifndef TARAXAGTESTS_PROCESSTEST_H
#define TARAXAGTESTS_PROCESSTEST_H

#include <string>
#include "Process.h"

using namespace std;

string TestBlindAuction() {
    // OK
    return RunTest(" --codefile clear_contracts/BlindAuction.bin --json run");
}

string TestAcknowledger() {
    // OK
    return RunTest(" --codefile clear_contracts/Acknowledger.bin --json run");
}

string TestCodeVerify() {
    // verify code
    return RunTest(" --codefile crash_contracts/unverify.bin --json run");
}

string TestVulnerabilityRecursive() {
    // crash vulnerability
    return RunTest(" --codefile crash_contracts/recursive.bin --json run");
}

string TestVulnerabilityLoop() {
    // crash vulnerability
    return RunTest(" --codefile crash_contracts/loop.bin --json run");
}

string TestGasOK() {
    // OK gas
    return RunTest(" --code 6040 --json run");
}

string TestOutOfGas() {
    // out of gas
    return RunTest(" --code 6040 --gas 0x1 --price 0x3 --json run");
}

#endif //TARAXAGTESTS_PROCESSTEST_H
