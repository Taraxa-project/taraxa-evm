// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/cmd/utils"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/core/vm/runtime"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/log"
	"google.golang.org/grpc"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
)

func runCmd(ctx *cli.Context) (ret []byte, leftOverGas uint64, err error) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.GlobalInt(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)
	logconfig := &vm.LogConfig{
		DisableMemory: ctx.GlobalBool(DisableMemoryFlag.Name),
		DisableStack:  ctx.GlobalBool(DisableStackFlag.Name),
		Debug:         ctx.GlobalBool(DebugFlag.Name),
	}
	var tracer vm.Tracer
	var debugLogger *vm.StructLogger
	if ctx.GlobalBool(MachineFlag.Name) {
		tracer = vm.NewJSONLogger(logconfig, os.Stdout)
	} else if logconfig.Debug {
		debugLogger = vm.NewStructLogger(logconfig)
		tracer = debugLogger
	} else {
		debugLogger = vm.NewStructLogger(logconfig)
	}
	isCreateOperation := ctx.GlobalBool(CreateFlag.Name)
	sender := common.HexToAddress(ctx.GlobalString(SenderFlag.Name))
	receiver := common.HexToAddress(ctx.GlobalString(ReceiverFlag.Name))
	dbAddress := ctx.GlobalString(RpcDatabaseAddressFlag.Name)
	conn, err := grpc.Dial(dbAddress, grpc.WithInsecure())
	if err != nil {
		return
	}
	contractAddress := receiver
	if isCreateOperation {
		contractAddress = sender
	}
	db := ethdb.NewRpcDatabase(conn, &ethdb.VmId{
		ContractAddress: contractAddress.Bytes(),
		ProcessId:       ctx.GlobalString(InstanceIdFlag.Name),
	})
	log.Info("Starting with rpc db, server address: " + dbAddress)
	stateRoot := common.HexToHash(ctx.GlobalString(StateRootFlag.Name))
	statedb, err := state.New(stateRoot, state.NewDatabase(db))
	if err != nil {
		return
	}
	statedb.CreateAccount(sender)
	input := common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))
	initialGas := ctx.GlobalUint64(GasFlag.Name)
	genesisConfig := new(core.Genesis)
	runtimeConfig := runtime.Config{
		Origin:      sender,
		State:       statedb,
		GasLimit:    initialGas,
		GasPrice:    utils.GlobalBig(ctx, PriceFlag.Name),
		Value:       utils.GlobalBig(ctx, ValueFlag.Name),
		Difficulty:  genesisConfig.Difficulty,
		Time:        new(big.Int).SetUint64(genesisConfig.Timestamp),
		Coinbase:    genesisConfig.Coinbase,
		BlockNumber: new(big.Int).SetUint64(genesisConfig.Number),
		EVMConfig: vm.Config{
			Tracer:         tracer,
			Debug:          logconfig.Debug || ctx.GlobalBool(MachineFlag.Name),
			EVMInterpreter: ctx.GlobalString(EVMInterpreterFlag.Name),
		},
	}
	if isCreateOperation {
		ret, _, leftOverGas, err = runtime.Create(input, &runtimeConfig)
	} else {
		ret, leftOverGas, err = runtime.Call(receiver, input, &runtimeConfig)
	}
	if err != nil {
		return
	}
	rootHash, err := Flush(statedb, nil)
	if err != nil {
		return
	}
	fmt.Println("stateRoot: " + rootHash.String())
	fmt.Println("ret:  " + common.Bytes2Hex(ret))
	if logconfig.Debug {
		if debugLogger != nil {
			fmt.Fprintln(os.Stderr, "#### TRACE ####")
			vm.WriteTrace(os.Stderr, debugLogger.StructLogs())
		}
		fmt.Fprintln(os.Stderr, "#### LOGS ####")
		vm.WriteLogs(os.Stderr, statedb.Logs())
	}
	return
}
