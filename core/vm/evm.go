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
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/dbg"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

// TODO OF TODOS: migrate away from big.Int to a fixed u256 library

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
type EVM struct {
	get_hash          GetHashFunc
	state             State
	block             Block
	rules             Rules
	rules_initialized bool
	precompiles       Precompiles
	instruction_set   InstructionSet
	gas_table         GasTable
	trx               *Transaction
	depth             uint16
	// tech stuff
	preallocated_stacks []Stack
	mem_pool            MemoryPool
	int_pool            int_pool
	bigconv             bigconv.BigConv
	jumpdests           map[common.Hash]bitvec // Aggregated result of JUMPDEST analysis.
	// call_gas_tmp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	call_gas_tmp uint64
	read_only    bool   // Whether to throw on stateful modifications
	last_retval  []byte // Last CALL's return data for subsequent reuse
}
type Opts = struct {
	U256PoolSize           uint32
	NumStacksToPreallocate uint16
	PreallocatedStackSize  uint16
	PreallocatedMem        uint64
}
type GetHashFunc = func(types.BlockNum) *big.Int
type Rules struct {
	IsHomestead, IsEIP150, IsEIP158, IsByzantium, IsConstantinople, IsPetersburg bool
}
type Block struct {
	Number types.BlockNum
	BlockInfo
}
type BlockInfo struct {
	Author     common.Address // Provides information for COINBASE
	GasLimit   uint64         // Provides information for GASLIMIT
	Time       uint64         // Provides information for TIME
	Difficulty *big.Int       // Provides information for DIFFICULTY
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
type ExecutionOpts struct {
	DisableNonceCheck, DisableGasFee bool
}
type ExecutionResult struct {
	CodeRetval      []byte
	NewContractAddr common.Address
	Logs            []LogRecord
	GasUsed         uint64
	CodeErr         util.ErrorString
	ConsensusErr    util.ErrorString
}

func (self *EVM) Init(get_hash GetHashFunc, state State, opts Opts) *EVM {
	assert.Holds(opts.NumStacksToPreallocate <= StackLimit)
	assert.Holds(opts.PreallocatedStackSize <= StackLimit)
	self.get_hash = get_hash
	self.state = state
	self.mem_pool.buf = make([]byte, opts.PreallocatedMem)
	self.int_pool.Init(int(opts.U256PoolSize))
	self.preallocated_stacks = make([]Stack, opts.NumStacksToPreallocate)
	for i := range self.preallocated_stacks {
		self.preallocated_stacks[i].Init(int(opts.PreallocatedStackSize))
	}
	return self
}

func (self *EVM) GetRules() Rules {
	return self.rules
}

func (self *EVM) GetDepth() uint16 {
	return self.depth
}

func (self *EVM) GetBlock() Block {
	return self.block
}

func (self *EVM) SetBlock(blk_num types.BlockNum, blk_info *BlockInfo, rules Rules) (rules_changed bool) {
	self.block.Number, self.block.BlockInfo = blk_num, *blk_info
	if self.rules_initialized {
		if self.rules == rules {
			return false
		}
	} else {
		self.rules_initialized = true
	}
	switch {
	case rules.IsConstantinople:
		self.precompiles = PrecompiledContractsByzantium
		self.instruction_set = constantinopleInstructionSet
		self.gas_table = GasTableConstantinople
	case rules.IsByzantium:
		self.precompiles = PrecompiledContractsByzantium
		self.instruction_set = byzantiumInstructionSet
		self.gas_table = GasTableEIP158
	case rules.IsEIP158:
		self.precompiles = PrecompiledContractsHomestead
		self.instruction_set = homesteadInstructionSet
		self.gas_table = GasTableEIP158
	case rules.IsEIP150:
		self.precompiles = PrecompiledContractsHomestead
		self.instruction_set = homesteadInstructionSet
		self.gas_table = GasTableEIP150
	case rules.IsHomestead:
		self.precompiles = PrecompiledContractsHomestead
		self.instruction_set = homesteadInstructionSet
		self.gas_table = GasTableHomestead
	default:
		self.precompiles = PrecompiledContractsHomestead
		self.instruction_set = frontierInstructionSet
		self.gas_table = GasTableHomestead
	}
	self.rules = rules
	return true
}

func (self *EVM) RegisterPrecompiledContract(address *common.Address, contract PrecompiledContract) {
	self.precompiles.Put(address, contract)
}

func (self *EVM) Main(trx *Transaction, opts ExecutionOpts) (ret ExecutionResult) {
	self.trx = trx
	defer func() { self.trx, self.jumpdests = nil, nil }()
	caller := self.state.GetAccount(&trx.From)
	if !opts.DisableNonceCheck {
		if nonce := caller.GetNonce(); nonce < self.trx.Nonce {
			ret.ConsensusErr = ErrNonceTooHigh
			return
		} else if nonce > self.trx.Nonce {
			ret.ConsensusErr = ErrNonceTooLow
			return
		}
	}
	gas_cap, gas_price, gas_fee := uint64(math.MaxUint64/2), bigutil.Big0, bigutil.Big0
	if !opts.DisableGasFee {
		gas_cap, gas_price = self.trx.Gas, self.trx.GasPrice
		gas_fee = new(big.Int).Mul(new(big.Int).SetUint64(gas_cap), gas_price)
	}
	gas_left := gas_cap
	contract_creation := self.trx.To == nil
	if !opts.DisableGasFee {
		if !BalanceGTE(caller, gas_fee) {
			ret.ConsensusErr = ErrInsufficientBalanceForGas
			return
		}
		caller.SubBalance(gas_fee)
		gas_intrinsic, err := IntrinsicGas(self.trx.Input, contract_creation, self.rules.IsHomestead)
		if err != nil {
			ret.ConsensusErr = util.ErrorString(err.Error())
			return
		}
		if gas_left < gas_intrinsic {
			ret.ConsensusErr = ErrIntrinsicGas
			return
		}
		gas_left -= gas_intrinsic
	}
	var err error
	if contract_creation {
		ret.CodeRetval, ret.NewContractAddr, gas_left, err = self.create_1(caller, self.trx.Input, gas_left, self.trx.Value)
	} else {
		acc_to := self.state.GetAccount(self.trx.To)
		caller.IncrementNonce()
		ret.CodeRetval, gas_left, err = self.call(caller, acc_to, self.trx.Input, gas_left, self.trx.Value)
	}
	if err != nil {
		if err_str := util.ErrorString(err.Error()); err_str == ErrInsufficientBalanceForTransfer {
			ret.ConsensusErr = err_str
			return
		} else {
			ret.CodeErr = err_str
		}
	}
	gas_left += util.Min_u64(self.state.GetRefund(), (gas_cap-gas_left)/2)
	ret.GasUsed = gas_cap - gas_left
	ret.Logs = self.state.GetLogs()
	if !opts.DisableGasFee {
		// Return ETH for remaining gas, exchanged at the original rate.
		caller.AddBalance(new(big.Int).Mul(new(big.Int).SetUint64(gas_left), gas_price))
		self.state.GetAccount(&self.block.Author).
			AddBalance(new(big.Int).Mul(new(big.Int).SetUint64(ret.GasUsed), gas_price))
	}
	return
}

// create_1 creates a new contract using code as deployment code.
func (self *EVM) create_1(caller StateAccount, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), caller.GetNonce())
	ret, leftOverGas, err = self.create(caller, CodeAndHash{Code: code}, gas, value, &contractAddr)
	return
}

// create_2 creates a new contract using code as deployment code.
//
// The different between create_2 with create_1 is create_2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (self *EVM) create_2(caller StateAccount, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	codeAndHash := CodeAndHash{code, keccak256.Hash(code)}
	contractAddr = crypto.CreateAddress2(caller.Address(), self.bigconv.ToHash(salt), codeAndHash.CodeHash[:])
	ret, leftOverGas, err = self.create(caller, codeAndHash, gas, endowment, &contractAddr)
	return
}

// create creates a new contract using code as deployment code.
func (self *EVM) create(
	caller StateAccount, code CodeAndHash, gas uint64, value *big.Int, address *common.Address) (
	ret []byte, gas_left uint64, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}
	if !BalanceGTE(caller, value) {
		return nil, gas, ErrInsufficientBalanceForTransfer
	}
	// TODO This should go after the state snapshot, but this is how it works in ETH
	caller.IncrementNonce()
	new_acc := self.state.GetAccount(address)
	// Ensure there's no existing contract already at the designated address
	if new_acc.GetNonce() != 0 || new_acc.GetCodeSize() != 0 {
		// TODO this also should check if new acc balance is zero, but this is how it works in ETH
		return nil, 0, ErrContractAddressCollision
	}
	// create a new account on the state
	snapshot := self.state.Snapshot()
	if self.rules.IsEIP158 {
		new_acc.IncrementNonce()
	}
	self.transfer(caller, new_acc, value)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(CallFrame{caller, new_acc, nil, gas, value}, code)
	ret, err = self.run(&contract, false)
	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := self.rules.IsEIP158 && len(ret) > MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * CreateDataGas
		if contract.UseGas(createDataGas) {
			new_acc.SetCode(ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (self.rules.IsHomestead || err != ErrCodeStoreOutOfGas)) {
		self.state.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	gas_left = contract.Gas
	return
}

// call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (self *EVM) call(caller, callee StateAccount, input []byte, gas uint64, value *big.Int) (ret []byte, gas_left uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}
	if value.Sign() == 0 {
		if self.rules.IsEIP158 && !callee.IsNotNIL() && self.precompiles.Get(callee.Address()) == nil {
			return nil, gas, nil
		}
	} else if !BalanceGTE(caller, value) {
		return nil, gas, ErrInsufficientBalanceForTransfer
	}
	snapshot := self.state.Snapshot()
	self.transfer(caller, callee, value)
	return self.call_end(CallFrame{caller, callee, input, gas, value}, callee, snapshot, false)
}

func (self *EVM) call_end(frame CallFrame, code_owner StateAccount, snapshot int, read_only bool) (ret []byte, gas_left uint64, err error) {
	gas_left = frame.Gas
	if precompiled := self.precompiles.Get(code_owner.Address()); precompiled != nil {
		if gas_required := precompiled.RequiredGas(frame, self); gas_required <= gas_left {
			gas_left -= gas_required
			ret, err = precompiled.Run(frame, self)
		} else {
			err = ErrOutOfGas
		}
	} else if code := code_owner.GetCode(); len(code) != 0 {
		contract := NewContract(frame, CodeAndHash{code, code_owner.GetCodeHash()})
		ret, err = self.run(&contract, read_only)
		gas_left = contract.Gas
	}
	if err != nil {
		self.state.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			gas_left = 0
		}
	}
	return
}

// call_code executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// call_code differs from call in the sense that it executes the given address'
// code with the caller as context.
func (self *EVM) call_code(caller *Contract, callee StateAccount, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !BalanceGTE(caller.Account, value) {
		return nil, gas, ErrInsufficientBalanceForTransfer
	}
	return self.call_end(CallFrame{caller.Account, caller.Account, input, gas, value}, callee, self.state.Snapshot(), false)
}

// call_delegate executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// call_delegate differs from call_code in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (self *EVM) call_delegate(caller *Contract, callee StateAccount, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}
	return self.call_end(CallFrame{caller.CallerAccount, caller.Account, input, gas, caller.Value}, callee, self.state.Snapshot(), false)
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (self *EVM) call_static(caller *Contract, callee StateAccount, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}
	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	snapshot := self.state.Snapshot()
	callee.AddBalance(bigutil.Big0)
	return self.call_end(CallFrame{caller.Account, callee, input, gas, bigutil.Big0}, callee, snapshot, true)
}

// loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// errExecutionReverted which means revert-and-keep-gas-left.
func (self *EVM) run(contract *Contract, readOnly bool) (ret []byte, err error) {
	var mem Memory
	mem.Init(&self.mem_pool)
	defer mem.Release()
	var stack *Stack
	if self.depth < uint16(len(self.preallocated_stacks)) {
		stack = &self.preallocated_stacks[self.depth]
		defer stack.reset()
	} else {
		stack = new(Stack).Init(StackLimit)
	}
	// Reclaim the stack as an int pool when the execution stops
	defer func() { self.int_pool.put(stack.data...) }()
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
		op OpCode // current opcode
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		res  []byte // result of the opcode execution function
	)
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
		cost, err = operation.gasCost(self, contract, stack, &mem, memorySize)
		if err != nil || !contract.UseGas(cost) {
			return nil, ErrOutOfGas
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}
		if dbg.Debug {
			fmt.Println("pc:", pc, "op:", op)
			stack.Print()
			mem.Print()
		}
		// execute the operation
		res, err = operation.execute(&pc, self, contract, &mem, stack)
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

func (self *EVM) analyze_jumpdests(code CodeAndHash) (analysis bitvec, cached bool) {
	if cached = code.CodeHash != nil; cached {
		if present := self.jumpdests != nil; !present {
			// TODO preallocate
			self.jumpdests = make(map[common.Hash]bitvec)
		} else if analysis, present = self.jumpdests[*code.CodeHash]; present {
			return
		}
		analysis = codeBitmap(code.Code)
		self.jumpdests[*code.CodeHash] = analysis
	} else {
		analysis = codeBitmap(code.Code)
	}
	return
}

func (self *EVM) transfer(from, to StateAccount, amount *big.Int) {
	from.SubBalance(amount)
	to.AddBalance(amount)
}

func (self *EVM) get_account(addr_as_big *big.Int) StateAccount {
	return self.state.GetAccount(self.bigconv.ToAddr(addr_as_big))
}
