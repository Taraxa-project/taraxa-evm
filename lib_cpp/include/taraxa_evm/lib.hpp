#ifndef TARAXA_EVM_LIB_HPP
#define TARAXA_EVM_LIB_HPP

#include <vector>
#include <type_traits>
#include <rapidjson/document.h>
#include "rapidjson/writer.h"
#include "rapidjson/stringbuffer.h"
#include <boost/multiprecision/cpp_int.hpp>
#include "cgo_bridge.hpp"

namespace taraxa_evm::__lib {
    using namespace std;
    using namespace rapidjson;
    using namespace boost::multiprecision;

    // From aleth
    using u256 =  number<cpp_int_backend<256, 256, unsigned_magnitude, unchecked, void>>;
    using HexString = string;
    using BigInt = u256;
    using Address = HexString;
    using Hash = HexString;
    using Bloom = string;

    struct Transaction {
        Address from;
        Address *to;
        uint64_t nonce;
        BigInt amount;
        uint64_t gasLimit;
        BigInt gasPrice;
        HexString *data;
    };

    struct Block {
        Address coinbase;
        BigInt number;
        BigInt difficulty;
        BigInt time;
        uint64_t gasLimit;
        Hash hash;
    };

    struct ConcurrentSchedule {
        vector<uint64_t> sequential;

        template<typename E, typename A>
        static ConcurrentSchedule fromJson(const GenericValue<E, A> &json) {
            ConcurrentSchedule concurrentSchedule{};
            for (auto &e : json["sequential"].GetArray()) {
                concurrentSchedule.sequential.emplace_back(e.GetUint64());
            }
            return concurrentSchedule;
        }

    };

    struct LDBConfig {
        string file;
        int32_t cache;
        int32_t handles;
    };

    struct RunConfiguration {
        Hash stateRoot;
        Block *block;
        vector<Transaction> *transactions;
        LDBConfig *ldbConfig;
        ConcurrentSchedule *concurrentSchedule;
    };

    struct Log {
        Address address;
        vector<Address> topics;
        HexString data;
        HexString blockNumber;
        Hash transactionHash;
        HexString transactionIndex;
        Hash blockHash;
        HexString index;
        bool removed;

        template<typename E, typename A>
        static Log fromJson(const GenericValue<E, A> &json) {
            Log log{};
            log.address = json["address"].GetString();
            for (auto &topic : json["topics"].GetArray()) {
                log.topics.emplace_back(topic.GetString());
            }
            log.data = json["data"].GetString();
            log.blockNumber = json["blockNumber"].GetString();
            log.transactionHash = json["transactionHash"].GetString();
            log.transactionIndex = json["transactionIndex"].GetString();
            log.blockHash = json["blockHash"].GetString();
            log.index = json["logIndex"].GetString();
            log.removed = json["removed"].GetBool();
            return log;
        }

    };

    struct Receipt {
        Hash root;
        HexString status;
        HexString cumulativeGasUsed;
        Bloom logsBloom;
        vector<Log> logs;
        Hash transactionHash;
        Address contractAddress;
        HexString gasUsed;

        template<typename E, typename A>
        static Receipt fromJson(const GenericValue<E, A> &json) {
            Receipt receipt{};
            receipt.root = json["root"].GetString();
            receipt.status = json["status"].GetString();
            receipt.cumulativeGasUsed = json["cumulativeGasUsed"].GetString();
            receipt.logsBloom = json["logsBloom"].GetString();
            auto &logs = json["logs"];
            if (!logs.IsNull()) {
                for (auto &log : logs.GetArray()) {
                    receipt.logs.emplace_back(Log::fromJson(log));
                }
            }
            receipt.transactionHash = json["transactionHash"].GetString();
            receipt.contractAddress = json["contractAddress"].GetString();
            receipt.gasUsed = json["gasUsed"].GetString();
            return receipt;
        }

    };

    struct Result {
        Hash stateRoot;
        ConcurrentSchedule concurrentSchedule;
        vector<Receipt> receipts;
        vector<Log> allLogs;
        uint64_t usedGas;
        vector<HexString> returnValues;
        string error;

        template<typename E, typename A>
        static Result fromJson(const GenericValue<E, A> &json) {
            Result result{};
            auto &err = json["error"];
            if (!err.IsNull()) {
                result.error = err.GetString();
                return result;
            }
            result.stateRoot = json["stateRoot"].GetString();
            auto &concurrentSchedule = json["concurrentSchedule"];
            if (!concurrentSchedule.IsNull()) {
                result.concurrentSchedule = ConcurrentSchedule::fromJson(concurrentSchedule);
            }
            for (auto &e : json["receipts"].GetArray()) {
                result.receipts.emplace_back(Receipt::fromJson(e));
            }
            auto &logs = json["allLogs"];
            if (!logs.IsNull()) {
                for (auto &e : json["allLogs"].GetArray()) {
                    result.allLogs.emplace_back(Log::fromJson(e));
                }
            }
            result.usedGas = json["usedGas"].GetInt64();
            for (auto &e : json["returnValues"].GetArray()) {
                result.returnValues.emplace_back(e.GetString());
            }
            return result;
        }

    };

    template<typename E, typename A>
    GenericValue<E, A> &set(Document &doc, GenericValue<E, A> &obj, const string &key, const Value &value) {
        obj.AddMember(StringRef(key.c_str()), const_cast<Value &>(value), doc.GetAllocator());
        return obj[key.c_str()];
    }

    template<typename E, typename A>
    GenericValue<E, A> &set(Document &doc, GenericValue<E, A> &obj, const string &key, const string &value) {
        return set(doc, obj, key, Value().SetString(StringRef(value.c_str())));
    }

    template<typename E, typename A, typename T, typename = typename
            enable_if<!(is_same<T, Value>::value || is_same<T, string>::value), T>::type>
    GenericValue<E, A> &set(Document &doc, GenericValue<E, A> &obj, const string &key, const T &value) {
        return set(doc, obj, key, Value().Set(value));
    }

    Result runEvm(const RunConfiguration &config) {
        Document configDoc;

        auto &rootObj = configDoc.SetObject();
        set(configDoc, rootObj, "stateRoot", config.stateRoot);
        if (config.concurrentSchedule) {
            auto &concurrentSchedule = set(configDoc, rootObj, "concurrentSchedule", Value());
            auto &sequential = set(configDoc, concurrentSchedule, "sequential", Value(kArrayType));
            for (auto &e : config.concurrentSchedule->sequential) {
                sequential.PushBack(e, configDoc.GetAllocator());
            }
        }
        auto &block = set(configDoc, rootObj, "block", Value(kObjectType));
        set(configDoc, block, "coinbase", config.block->coinbase);
        set(configDoc, block, "hash", config.block->hash);
        set(configDoc, block, "number", config.block->number.str());
        set(configDoc, block, "difficulty", config.block->difficulty.str());
        set(configDoc, block, "time", config.block->time.str());
        set(configDoc, block, "gasLimit", config.block->gasLimit);
        auto &transactions = set(configDoc, rootObj, "transactions", Value()).SetArray();
        for (auto &e : *config.transactions) {
            Value transaction(kObjectType);
            if (e.to) {
                set(configDoc, transaction, "to", *e.to);
            }
            if (e.data) {
                set(configDoc, transaction, "data", *e.data);
            }
            set(configDoc, transaction, "from", e.from);
            set(configDoc, transaction, "gasLimit", e.gasLimit);
            set(configDoc, transaction, "gasPrice", e.gasPrice.str());
            set(configDoc, transaction, "nonce", e.nonce);
            set(configDoc, transaction, "amount", e.amount.str());
            transactions.PushBack(transaction, configDoc.GetAllocator());
        }
        auto &ldbConfig = set(configDoc, rootObj, "ldbConfig", Value()).SetObject();
        set(configDoc, ldbConfig, "file", config.ldbConfig->file);
        set(configDoc, ldbConfig, "handles", config.ldbConfig->handles);
        set(configDoc, ldbConfig, "cache", config.ldbConfig->cache);

        StringBuffer buffer;
        Writer<StringBuffer> writer(buffer);
        configDoc.Accept(writer);
        auto resultStr = cgo_bridge::runEvm(buffer.GetString());
        Document resultDoc;
        resultDoc.Parse(resultStr.c_str());
        return Result::fromJson(resultDoc);
    }

}
namespace taraxa_evm::lib {
    using __lib::runEvm;
    using __lib::Log;
    using __lib::ConcurrentSchedule;
    using __lib::Block;
    using __lib::Transaction;
    using __lib::LDBConfig;
    using __lib::Receipt;
    using __lib::RunConfiguration;
    using __lib::Result;
    using __lib::HexString;
    using __lib::Address;
    using __lib::BigInt;
    using __lib::Hash;
    using __lib::Bloom;
}

#endif //TARAXA_EVM_LIB_HPP