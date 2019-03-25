extern "C" {
#include "vm.h"
}

#include <string>

using namespace std;

int main(int argc, char **argv) {
    string s = "ololo";
    long size = s.length();
    auto str = GoString{s.c_str(), size};
    ProcessJson(str);
}