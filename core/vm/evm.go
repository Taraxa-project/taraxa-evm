// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
	"sync"
)

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
type EVM struct {
	*EVMConfig
	state State
	trx   *Transaction
	depth int
	// call_gas_tmp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	call_gas_tmp uint64
	int_pool     *int_pool
	read_only    bool   // Whether to throw on stateful modifications
	last_retval  []byte // Last CALL's return data for subsequent reuse
}
type EVMConfig struct {
	get_hash        GetHashFunc
	opts            ExecutionOptions
	rules           params.Rules
	precompiles     Precompiles
	instruction_set *InstructionSet
	gas_table       GasTable
	blk             *Block
}
type ExecutionOptions struct {
	DisableNonceCheck, DisableGasFee bool
}
type GetHashFunc = func(types.BlockNum) *big.Int
type Precompiles = map[common.Address]PrecompiledContract
type InstructionSet = [256]operation
type BlockWithoutNumber struct {
	Author     common.Address // Provides information for COINBASE
	GasLimit   uint64         // Provides information for GASLIMIT
	Time       uint64         // Provides information for TIME
	Difficulty *big.Int       // Provides information for DIFFICULTY
}
type Block struct {
	Number types.BlockNum // Provides information for NUMBER
	BlockWithoutNumber
}
type Transaction struct {
	From     common.Address  // Provides information for ORIGIN
	GasPrice *big.Int        // Provides information for GASPRICE
	To       *common.Address `rlp:"nil"`
	Nonce    uint64
	Value    *big.Int
	Gas      uint64
	Input    []byte
}

func NewEVMConfig(get_hash GetHashFunc, blk *Block, rules params.Rules, opts ExecutionOptions) (ret EVMConfig) {
	ret.get_hash = get_hash
	ret.opts = opts
	ret.rules = rules
	ret.blk = blk
	switch {
	case rules.IsConstantinople:
		ret.precompiles = PrecompiledContractsByzantium
		ret.instruction_set = &constantinopleInstructionSet
		ret.gas_table = GasTableConstantinople
	case rules.IsByzantium:
		ret.precompiles = PrecompiledContractsByzantium
		ret.instruction_set = &byzantiumInstructionSet
		ret.gas_table = GasTableEIP158
	case rules.IsEIP158:
		ret.precompiles = PrecompiledContractsHomestead
		ret.instruction_set = &homesteadInstructionSet
		ret.gas_table = GasTableEIP158
	case rules.IsEIP150:
		ret.precompiles = PrecompiledContractsHomestead
		ret.instruction_set = &homesteadInstructionSet
		ret.gas_table = GasTableEIP150
	case rules.IsHomestead:
		ret.precompiles = PrecompiledContractsHomestead
		ret.instruction_set = &homesteadInstructionSet
		ret.gas_table = GasTableHomestead
	default:
		ret.precompiles = PrecompiledContractsHomestead
		ret.instruction_set = &frontierInstructionSet
		ret.gas_table = GasTableHomestead
	}
	return ret
}

type ExecutionResult struct {
	CodeRet         []byte
	NewContractAddr common.Address
	Logs            []LogRecord
	GasUsed         uint64
	CodeErr         util.ErrorString
	ConsensusErr    util.ErrorString
}

func Main(cfg *EVMConfig, state State, trx *Transaction) (ret ExecutionResult) {
	if !cfg.opts.DisableNonceCheck {
		if nonce := state.GetNonce(trx.From); nonce < trx.Nonce {
			ret.ConsensusErr = ErrNonceTooHigh
			return
		} else if nonce > trx.Nonce {
			ret.ConsensusErr = ErrNonceTooLow
			return
		}
	}
	gas_cap, gas_price := trx.Gas, trx.GasPrice
	if cfg.opts.DisableGasFee {
		gas_cap, gas_price = ^uint64(0)/100000, common.Big0
	}
	gas_fee := new(big.Int).Mul(new(big.Int).SetUint64(gas_cap), gas_price)
	gas_left := gas_cap
	contract_creation := trx.To == nil
	if !cfg.opts.DisableGasFee {
		if !state.AssertBalanceGTE(trx.From, gas_fee) {
			ret.ConsensusErr = ErrInsufficientBalanceForGas
			return
		}
		if gas_intrinsic, err := IntrinsicGas(trx.Input, contract_creation, cfg.rules.IsHomestead); err != nil {
			ret.ConsensusErr = util.ErrorString(err.Error())
			return
		} else {
			if gas_left < gas_intrinsic {
				ret.ConsensusErr = ErrOutOfGas
				return
			}
			gas_left -= gas_intrinsic
		}
	}
	if !state.AssertBalanceGTE(trx.From, trx.Value) {
		ret.ConsensusErr = ErrInsufficientBalance
		return
	}
	if !cfg.opts.DisableGasFee {
		state.SubBalance(trx.From, gas_fee)
	}
	var run_code func(*EVM) error
	if contract_creation {
		run_code = func(evm *EVM) (err error) {
			ret.CodeRet, ret.NewContractAddr, gas_left, err =
				evm.create_1(AccountRef(trx.From), trx.Input, gas_left, trx.Value)
			return
		}
	} else {
		state.IncrementNonce(trx.From)
		contract, snapshot := call_begin(cfg, state, AccountRef(trx.From), *trx.To, trx.Input, gas_left, trx.Value)
		if contract != nil {
			run_code = func(evm *EVM) (err error) {
				ret.CodeRet, gas_left, err = evm.call_end(contract, snapshot, false)
				return
			}
		}
	}
	if run_code != nil {
		if err := run_code(&EVM{EVMConfig: cfg, state: state, trx: trx}); err != nil {
			ret.CodeErr = util.ErrorString(err.Error())
		}
		ret.Logs = state.GetLogs()
		if refund, refund_max := state.GetRefund(), (gas_cap-gas_left)/2; refund < refund_max {
			gas_left += refund
		} else {
			gas_left += refund_max
		}
	}
	ret.GasUsed = gas_cap - gas_left
	if !cfg.opts.DisableGasFee {
		// Return ETH for remaining gas, exchanged at the original rate.
		state.AddBalance(trx.From, new(big.Int).Mul(new(big.Int).SetUint64(gas_left), gas_price))
		state.AddBalance(cfg.blk.Author, new(big.Int).Mul(new(big.Int).SetUint64(ret.GasUsed), gas_price))
	}
	return
}

// call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) call(caller ContractRef, callee common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	if !evm.state.AssertBalanceGTE(caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}
	contract, snapshot := call_begin(evm.EVMConfig, evm.state, caller, callee, input, gas, value)
	if contract != nil {
		return evm.call_end(contract, snapshot, false)
	}
	return nil, gas, nil
}

func call_begin(cfg *EVMConfig, state State, caller ContractRef, callee common.Address, input []byte, gas uint64, value *big.Int) (contract *Contract, snapshot int) {
	var code []byte
	precompiled, is_precompiled := cfg.precompiles[callee]
	if !is_precompiled {
		if !state.Exist(callee) {
			if cfg.rules.IsEIP158 && value.Sign() == 0 {
				return
			}
		} else {
			code = state.GetCode(callee)
		}
	}
	if is_precompiled || len(code) != 0 {
		contract, snapshot = NewContract(caller, AccountRef(callee), value, gas, input), state.Snapshot()
		if is_precompiled {
			contract.precompiled = precompiled
		} else {
			contract.SetCallCode(state.GetCodeHash(callee), code)
		}
	}
	transfer(state, caller.Address(), callee, value)
	return
}

func (evm *EVM) call_end(contract *Contract, snapshot int, read_only bool) (ret []byte, gas_left uint64, err error) {
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if ret, err = evm.run(contract, read_only); err != nil {
		evm.state.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// create_1 creates a new contract using code as deployment code.
func (evm *EVM) create_1(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), evm.state.GetNonce(caller.Address()))
	return evm.create(caller, codeAndHash{code: code}, gas, value, contractAddr)
}

// create_2 creates a new contract using code as deployment code.
//
// The different between create_2 with create_1 is create_2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (evm *EVM) create_2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	codeAndHash := codeAndHash{code, util.HashOnStack(code)}
	contractAddr = crypto.CreateAddress2(caller.Address(), common.BigToHash(salt), codeAndHash.hash[:])
	return evm.create(caller, codeAndHash, gas, endowment, contractAddr)
}

type codeAndHash struct {
	code []byte
	hash common.Hash
}

// create creates a new contract using code as deployment code.
func (evm *EVM) create(caller ContractRef, codeAndHash codeAndHash, gas uint64, value *big.Int, address common.Address) ([]byte, common.Address, uint64, error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if evm.depth != 0 && !evm.state.AssertBalanceGTE(caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	evm.state.IncrementNonce(caller.Address())
	// Ensure there's no existing contract already at the designated address
	if evm.state.GetNonce(address) != 0 || evm.state.GetCodeSize(address) != 0 {
		panic("not sure")
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// create a new account on the state
	snapshot := evm.state.Snapshot()
	if evm.rules.IsEIP158 {
		evm.state.IncrementNonce(address)
	}
	transfer(evm.state, caller.Address(), address, value)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(address), value, gas, nil)
	contract.SetCallCode(codeAndHash.hash, codeAndHash.code)
	ret, err := evm.run(contract, false)
	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := evm.rules.IsEIP158 && len(ret) > MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.state.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evm.rules.IsHomestead || err != ErrCodeStoreOutOfGas)) {
		evm.state.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	return ret, address, contract.Gas, err
}

// call_code executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// call_code differs from call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) call_code(caller ContractRef, callee common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.state.AssertBalanceGTE(caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}
	snapshot := evm.state.Snapshot()
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(caller.Address()), value, gas, input)
	if contract.precompiled = evm.precompiles[callee]; contract.precompiled == nil {
		contract.SetCallCode(evm.state.GetCodeHash(callee), evm.state.GetCode(callee))
	}
	return evm.call_end(contract, snapshot, false)
}

// call_delegate executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// call_delegate differs from call_code in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) call_delegate(caller ContractRef, callee common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	snapshot := evm.state.Snapshot()
	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, AccountRef(caller.Address()), nil, gas, input).AsDelegate()
	if contract.precompiled = evm.precompiles[callee]; contract.precompiled == nil {
		contract.SetCallCode(evm.state.GetCodeHash(callee), evm.state.GetCode(callee))
	}
	return evm.call_end(contract, snapshot, false)
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) call_static(caller ContractRef, callee common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	snapshot := evm.state.Snapshot()
	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	evm.state.AddBalance(callee, bigZero)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(callee), new(big.Int), gas, input)
	if contract.precompiled = evm.precompiles[callee]; contract.precompiled == nil {
		contract.SetCallCode(evm.state.GetCodeHash(callee), evm.state.GetCode(callee))
	}
	return evm.call_end(contract, snapshot, true)
}

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func (evm *EVM) run(contract *Contract, readOnly bool) ([]byte, error) {
	if contract.precompiled != nil {
		return RunPrecompiledContract(contract)
	}
	return evm.run_code(contract, readOnly)
}

var stack_pool = sync.Pool{New: func() interface{} { return newstack() }}

// loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// errExecutionReverted which means revert-and-keep-gas-left.
func (self *EVM) run_code(contract *Contract, readOnly bool) (ret []byte, err error) {
	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return
	}
	// TODO don't release so often
	if self.int_pool == nil {
		self.int_pool = poolOfIntPools.get()
		defer func() {
			poolOfIntPools.put(self.int_pool)
			self.int_pool = nil
		}()
	}
	// Increment the call depth which is restricted to 1024
	self.depth++
	defer func() { self.depth-- }()
	// Make sure the read_only is only set if we aren't in read_only yet.
	// This makes also sure that the read_only flag isn't removed for child calls.
	if readOnly && !self.read_only {
		self.read_only = true
		defer func() { self.read_only = false }()
	}
	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	self.last_retval = nil
	var (
		op  OpCode        // current opcode
		mem = NewMemory() // bound memory
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		res  []byte // result of the opcode execution function
	)
	stack := stack_pool.Get().(*stack)
	defer func() {
		stack.data = stack.data[:0]
		stack_pool.Put(stack)
	}()
	// Reclaim the stack as an int pool when the execution stops
	defer func() { self.int_pool.put(stack.data...) }()
	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := self.instruction_set[op]
		if !operation.valid {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}
		if err = operation.validateStack(stack); err != nil {
			return nil, err
		}
		// If the operation is valid, enforce and write restrictions
		if self.rules.IsByzantium {
			if self.read_only {
				// If the interpreter is operating in readonly mode, make sure no
				// state-modifying operation is performed. The 3rd stack item
				// for a call operation is the value. Transferring value from one
				// account to the others means the state is modified and should also
				// return with an error.
				if operation.writes || (op == CALL && stack.Back(2).BitLen() > 0) {
					return nil, errWriteProtection
				}
			}
		}
		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		if operation.memorySize != nil {
			memSize, overflow := bigUint64(operation.memorySize(stack))
			if overflow {
				return nil, errGasUintOverflow
			}
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
				return nil, errGasUintOverflow
			}
		}
		// consume the gas and return an error if not enough gas is available.
		// cost is explicitly set so that the capture state defer method can get the proper cost
		cost, err = operation.gasCost(self, contract, stack, mem, memorySize)
		if err != nil || !contract.UseGas(cost) {
			return nil, ErrOutOfGas
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}
		// execute the operation
		res, err = operation.execute(&pc, self, contract, mem, stack)
		// if the operation clears the return data (e.g. it has returning data)
		// set the last return to the result of the operation.
		if operation.returns {
			self.last_retval = res
		}
		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return res, errExecutionReverted
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}
	}
	return nil, nil
}

func transfer(state State, from, to common.Address, amount *big.Int) {
	state.SubBalance(from, amount)
	state.AddBalance(to, amount)
}
