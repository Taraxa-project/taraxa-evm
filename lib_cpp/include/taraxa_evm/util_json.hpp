#ifndef TARAXA_EVM_GRPC_UTIL_JSON
#define TARAXA_EVM_GRPC_UTIL_JSON

#include <functional>
#include <stdexcept>
#include <rapidjson/document.h>
#include "taraxa_evm/util_str.hpp"

namespace taraxa_evm::__util_json {

    using namespace std;
    using namespace rapidjson;
    using namespace taraxa::util_str;

    template<typename V, typename K, typename E, typename A>
    V getField(const GenericValue<E, A> &root,
               const K &key,
               const V &defaultValue,
               const function<V(decltype(root))> &decoder) {
        if (root.IsNull()) return defaultValue;
        GenericValue<E, A> genericKey(key);
        if (root.IsObject())
            return root.HasMember(genericKey) ? decoder(root[key]) : defaultValue;
        else if (root.IsArray()) {
            auto keyInt = genericKey.GetInt();
            if (keyInt < 0) throw invalid_argument(fmt("index %s < 0", keyInt));
            return keyInt < root.GetArray().Size() ? decoder(root[keyInt]) : defaultValue;
        }
        throw invalid_argument("Attempted to indirect a primitive type");
    }

    template<typename K, typename E, typename A>
    int get(const GenericValue<E, A> &root, const K &key, const int &defaultValue = 0) {
        return getField<int>(root, key, defaultValue, [](auto &field) {
            return field.GetInt();
        });
    }

    template<typename K, typename E, typename A>
    string get(const GenericValue<E, A> &root, const K &key, const string &defaultValue = "") {
        return getField<string>(root, key, defaultValue, [](auto &field) {
            return field.GetString();
        });
    }

}
namespace taraxa_evm::util_json {
    using __util_json::get;
}

#endif //TARAXA_EVM_GRPC_UTIL_JSON
