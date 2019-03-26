#ifndef TARAXA_EVM_LIB_HPP
#define TARAXA_EVM_LIB_HPP

#include <vector>
//#include <boost/format.hpp>
#include "cgo_bridge.hpp"
#include <rapidjson/document.h>
#include "rapidjson/writer.h"
#include "rapidjson/stringbuffer.h"


namespace taraxa_evm::__lib {
    using namespace std;
    using namespace rapidjson;

    using HexString = string;
    using BigIntString = string;
    using Address = HexString;
    using Hash = HexString;
    using Bloom = string;

    struct Transaction {
        Address *from;
        Address *to;
        uint64_t nonce;
        BigIntString *amount;
        uint64_t gasLimit;
        BigIntString *gasPrice;
        HexString *data;
    };

    struct Block {
        Address *coinbase;
        BigIntString *number;
        BigIntString *difficulty;
        BigIntString *time;
        uint64_t gasLimit;
        Hash *hash;
    };

    struct ConcurrentSchedule {
        vector<uint64_t> sequential;
    };

    struct LDBConfig {
        string *file;
        int32_t cache;
        int32_t handles;
    };

    struct Log {
        Address address;
        vector<Address> topics;
        HexString data;
        uint64_t blockNumber;
        Hash txHash;
        uint32_t txIndex;
        Hash blockHash;
        uint32_t index;
        bool removed;
    };

    struct Receipt {
        Hash *postState;
        uint64_t status;
        uint64_t cumulativeGasUsed;
        Bloom *logsBloom;
        vector<Log> *logs;
        Hash *transactionHash;
        Address *contractAddress;
        uint64_t gasUsed;
    };

    struct RunConfiguration {
        Hash *stateRoot;
        Block *block;
        vector<Transaction> *transactions;
        LDBConfig *ldbConfig;
        ConcurrentSchedule *concurrentSchedule;
    };

    struct Result {
        Hash stateRoot;
        ConcurrentSchedule concurrentSchedule;
        vector<Receipt> receipts;
        vector<Log> allLogs;
        uint64_t usedGas;
        vector<HexString> returnValues;
        string error;
    };

    Result run(const RunConfiguration &config) {
        // TODO
//        Document configDoc;
//
//        auto &rootObj = configDoc.SetObject();
//        rootObj["stateRoot"] = config->stateRoot.c_str();
//
//        if (config.concurrentSchedule) {
//            auto &concurrentSchedule = rootObj["concurrentSchedule"].SetObject();
//            auto &sequential = concurrentSchedule["sequential"].SetArray();
//            for (auto &e : config.concurrentSchedule->sequential) {
//                sequential.PushBack(e, configDoc.GetAllocator());
//            }
//        }
//
//        auto &block = rootObj["block"].SetObject();
//        block["coinbase"] = *config.block->coinbase;
//        block["number"] = *config.block->number;
//        block["difficulty"] = *config.block->difficulty;
//        block["time"] = *config.block->time;
//        block["gasLimit"] = config.block->gasLimit;
//        block["hash"] = *config.block->hash;
//
//        auto &transactions = rootObj["transactions"].SetArray();
//        for (auto &e : *config.transactions) {
//            Value transaction;
//            transaction["from"] = *e.from;
//            transaction["to"] = *e.to;
//            transaction["gasLimit"] = e.gasLimit;
//            transaction["gasPrice"] = *e.gasPrice;
//            transaction["nonce"] = e.nonce;
//            transaction["amount"] = *e.amount;
//            transaction["data"] = *e.data;
//            transactions.PushBack(transaction, configDoc.GetAllocator());
//        }
//
//        auto &ldbConfig = rootObj["ldbConfig"].SetObject();
//        ldbConfig["file"] = *config.ldbConfig->file;
//        ldbConfig["handles"] = config.ldbConfig->handles;
//        ldbConfig["cache"] = config.ldbConfig->cache;
//
//        StringBuffer buffer;
//        Writer<StringBuffer> writer(buffer);
//        configDoc.Accept(writer);
//
//        auto resultStr = cgo_bridge::run(buffer.GetString());
//
//        Document resultDoc;
//        resultDoc.Parse(resultStr.c_str());
//
//        Result result{};
//
//        auto &err = resultDoc["error"];
//        if (!err.IsNull()) {
//            result.error = err.GetString();
//            return result;
//        }
//        result.stateRoot = resultDoc["stateRoot"].GetString();
//        result.usedGas = resultDoc["usedGas"].GetInt64();
//        auto &concurrentSchedule = resultDoc["concurrentSchedule"];
//        if (!concurrentSchedule.IsNull()) {
//            for (auto &e : concurrentSchedule.GetArray()) {
//                result.concurrentSchedule.sequential.push_back(e.GetUint64());
//            }
//        }
//        for (auto &log : resultDoc["allLogs"].GetArray()) {
//            result.allLogs.push_back(Log{
//                    .address = log["address"].GetString()
//            });
//        }
        Result result{};
        return result;
    }


}
namespace taraxa_evm::lib {
    using __lib::run;
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
    using __lib::BigIntString;
    using __lib::Hash;
    using __lib::Bloom;
}

#endif //TARAXA_EVM_LIB_HPP