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
	"time"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/holiman/uint256"
)

// Config are the configuration options for the Interpreter
type Config struct {
	// Debug enabled debugging Interpreter options
	Debug bool
	// Tracer is the op code logger
	Tracer Tracer
}

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
type EVM struct {
	get_hash GetHashFunc
	state    State
	block    Block
	// virtual machine configuration options used to initialize the evm
	vmConfig          Config
	chainConfig       params.ChainConfig
	rules             Rules
	rules_initialized bool
	precompiles       Precompiles
	instruction_set   InstructionSet
	gas_table         GasTable
	trx               *Transaction
	depth             uint16
	// tech stuff
	mem_pool  MemoryPool
	bigconv   bigconv.BigConv
	jumpdests map[common.Hash]bitvec // Aggregated result of JUMPDEST analysis.
	// call_gas_tmp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	call_gas_tmp uint64
	read_only    bool   // Whether to throw on stateful modifications
	last_retval  []byte // Last CALL's return data for subsequent reuse
}
type Opts = struct {
	PreallocatedMem uint64
}

func DefaultOpts() Opts {
	return Opts{
		PreallocatedMem: 8 * 1024 * 1024,
	}
}

type GetHashFunc = func(types.BlockNum) *big.Int

type Rules struct {
	IsMagnolia     bool
	IsAspenPartOne bool
	IsAspenPartTwo bool
	IsFicus        bool
	IsCornus       bool
	IsCacti        bool
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
	Nonce    *big.Int
	Value    *big.Int
	Gas      uint64
	Input    []byte
}
type ExecutionResult struct {
	CodeRetval      []byte
	NewContractAddr common.Address
	Logs            []LogRecord
	GasUsed         uint64
	ExecutionErr    util.ErrorString
	ConsensusErr    util.ErrorString
}

func (self *EVM) Init(get_hash GetHashFunc, state State, opts Opts, chainConfig params.ChainConfig, vmConfig Config) *EVM {
	self.get_hash = get_hash
	self.state = state
	self.chainConfig = chainConfig
	self.vmConfig = vmConfig
	self.mem_pool.Init(opts.PreallocatedMem)
	return self
}

func (self *EVM) UpdateVmConfig(vmConfig Config) {
	self.vmConfig = vmConfig
}

func (self *EVM) AddLog(log LogRecord) {
	self.state.AddLog(log)
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

func (self *EVM) SetBlock(blk *Block, rules Rules) (rules_changed bool) {
	self.block = *blk
	if self.rules_initialized {
		if self.rules == rules {
			return false
		}
	} else {
		self.rules_initialized = true
	}
	// Copy global precompile maps into an instance-specific map to avoid
	// concurrent reads/writes across different EVM instances.
	copyPrecompiles := func(src Precompiles) Precompiles {
		dst := make(Precompiles, len(src))
		for k, v := range src {
			dst[k] = v
		}
		return dst
	}

	switch {
	case rules.IsCacti:
		self.precompiles = copyPrecompiles(PrecompiledContractsCacti)
		self.instruction_set = cactiInstructionSet
		self.gas_table = GasTableCalifornicum
	case rules.IsFicus:
		self.precompiles = copyPrecompiles(PrecompiledContractsFicus)
		self.instruction_set = ficusInstructionSet
		self.gas_table = GasTableCalifornicum
	default:
		self.precompiles = copyPrecompiles(PrecompiledContractsCalifornicum)
		self.instruction_set = californicumInstructionSet
		self.gas_table = GasTableCalifornicum
	}
	self.rules = rules
	return true
}

func (self *EVM) RegisterPrecompiledContract(address *common.Address, contract PrecompiledContract) {
	self.precompiles.Put(address, contract)
}

func consensusErr(result ExecutionResult, trxGas uint64, consensusErr error) (ret ExecutionResult, err error) {
	ret = result
	ret.GasUsed = trxGas
	ret.ConsensusErr = util.ErrorString(consensusErr.Error())
	return
}

func (self *EVM) Main(trx *Transaction) (ret ExecutionResult, execError error) {
	self.trx = trx

	defer func() { self.trx, self.jumpdests = nil, nil }()

	caller := self.state.GetAccount(&self.trx.From)
	sender_nonce := caller.GetNonce()

	gas_cap, gas_price := self.trx.Gas, self.trx.GasPrice
	gas_fee := new(big.Int).Mul(new(big.Int).SetUint64(gas_cap), gas_price)
	contract_creation := self.trx.To == nil

	if self.trx.From != common.ZeroAddress && !BalanceGTE(caller, gas_fee) {
		caller_balance := caller.GetBalance()
		// Not Sub whole balance to have transaction sender balance difference match `gas_used * gas_price`. So balance after SubBalance should be smaller than gas_price
		available_funds_gas := bigutil.Div(caller_balance, gas_price)
		caller.SubBalance(bigutil.Mul(available_funds_gas, gas_price))

		if self.rules.IsCornus && self.trx.Nonce.Cmp(sender_nonce) >= 0 {
			caller.SetNonce(bigutil.Add(self.trx.Nonce, big.NewInt(1)))
		}
		return consensusErr(ret, available_funds_gas.Uint64(), ErrInsufficientBalanceForGas)
	}

	caller.SubBalance(gas_fee)

	// Check if tx.nonce < sender_nonce
	if self.trx.Nonce.Cmp(sender_nonce) < 0 {
		return consensusErr(ret, gas_cap, ErrNonceTooLow)
	}

	// Nonce skipping is permanently enabled now. Uncomment this part to have strict nonce ordering
	// Check if tx.nonce > sender_nonce + 1
	// if self.trx.Nonce.Cmp(bigutil.Add(sender_nonce, big.NewInt(1))) > 0 {
	// 	ret.ConsensusErr = ErrNonceTooHigh
	// 	return
	// }

	gas_intrinsic, err := IntrinsicGas(self.trx.Input, contract_creation)
	if err != nil {
		if self.rules.IsCornus {
			caller.SetNonce(bigutil.Add(self.trx.Nonce, big.NewInt(1)))
		}
		return consensusErr(ret, gas_cap, err)
	}

	if gas_cap < gas_intrinsic {
		if self.rules.IsCornus {
			caller.SetNonce(bigutil.Add(self.trx.Nonce, big.NewInt(1)))
		}
		return consensusErr(ret, gas_cap, ErrIntrinsicGas)
	}

	gas_left := gas_cap
	gas_left -= gas_intrinsic

	if contract_creation {
		// setting nonce to current trx nonce to generate correct address for new contract. Nonce incremented inside of `create` later
		caller.SetNonce(self.trx.Nonce)
		ret.CodeRetval, ret.NewContractAddr, gas_left, err = self.create_1(caller, self.trx.Input, gas_left, self.trx.Value)
	} else {
		acc_to := self.state.GetAccount(self.trx.To)
		caller.SetNonce(bigutil.Add(self.trx.Nonce, big.NewInt(1)))
		ret.CodeRetval, gas_left, err = self.Call(ContractAccWrapper{caller}, acc_to, self.trx.Input, gas_left, self.trx.Value)
	}
	if err != nil {
		if err == ErrInsufficientBalanceForTransfer {
			return consensusErr(ret, gas_cap, err)
		} else {
			ret.ExecutionErr = util.NewErrorString(err)
			execError = err
		}
	}
	gas_left += util.MinU64(self.state.GetRefund(), (gas_cap-gas_left)/2)
	ret.GasUsed = gas_cap - gas_left
	ret.Logs = self.state.GetLogs()
	// Return ETH for remaining gas, exchanged at the original rate.
	caller.AddBalance(new(big.Int).Mul(new(big.Int).SetUint64(gas_left), gas_price))
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
// The different between create_2 with create_1 is create_2 uses keccak256(0xff ++ msg.sender ++ salt ++ keccak256(init_code))[12:]
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
	if *caller.Address() != common.ZeroAddress && !BalanceGTE(caller, value) {
		return nil, gas, ErrInsufficientBalanceForTransfer
	}
	// TODO This should go after the state snapshot, but this is how it works in ETH
	caller.IncrementNonce()
	new_acc := self.state.GetAccount(address)
	// Ensure there's no existing contract already at the designated address
	if new_acc.GetNonce().Sign() != 0 || new_acc.GetCodeSize() != 0 {
		// TODO this also should check if new acc balance is zero, but this is how it works in ETH
		return nil, 0, ErrContractAddressCollision
	}
	// create a new account on the state
	snapshot := self.state.Snapshot()
	new_acc.IncrementNonce()
	self.transfer(caller, new_acc, value)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(CallFrame{caller, new_acc, nil, gas, value}, code)
	if self.vmConfig.Debug {
		if self.depth == 0 {
			self.vmConfig.Tracer.CaptureStart(self, caller.Address(), address, false /* precompile */, true /* create */, code.Code, gas, value, nil)
		} else {
			self.vmConfig.Tracer.CaptureEnter(CREATE, caller.Address(), address, false /* precompile */, true /* create */, code.Code, gas, value, nil)
		}
	}
	start := time.Now()

	ret, err = self.run(&contract, false)

	// Check whether the max code size has been exceeded, assign err if the case.
	if err == nil && len(ret) > MaxCodeSize {
		err = ErrMaxCodeSizeExceeded
	}

	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil {
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
	if err != nil {
		self.state.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	if self.vmConfig.Debug {
		if self.depth == 0 {
			self.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		} else {
			self.vmConfig.Tracer.CaptureExit(ret, gas-contract.Gas, time.Since(start), err)
		}
	}
	gas_left = contract.Gas
	return
}

// call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (self *EVM) call(typ OpCode, caller ContractRef, callee StateAccount, input []byte, gas uint64, value *big.Int) (ret []byte, gas_left uint64, err error) {
	// Fail if we're trying to execute above the call depth limit
	if self.depth > CallCreateDepth {
		return nil, gas, ErrDepth
	}

	if typ == CALL || typ == CALLCODE {
		if *caller.Address() != common.ZeroAddress && !BalanceGTE(caller.Account(), value) {
			return nil, gas, ErrInsufficientBalanceForTransfer
		}
	}

	snapshot := self.state.Snapshot()

	if typ == CALL {
		if value.Sign() == 0 {
			if callee.IsNIL() && self.precompiles.Get(callee.Address()) == nil {
				// Calling a non existing account, don't do anything, but ping the tracer
				if self.vmConfig.Debug {
					if self.depth == 0 {
						self.vmConfig.Tracer.CaptureStart(self, caller.Address(), callee.Address(), false, false, input, gas, value, nil)
						self.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
					} else {
						self.vmConfig.Tracer.CaptureEnter(CALL, caller.Address(), callee.Address(), false, false, input, gas, value, nil)
						self.vmConfig.Tracer.CaptureExit(ret, 0, 0, nil)
					}
				}
				return nil, gas, nil
			}
		}
		self.transfer(caller.Account(), callee, value)
	} else if typ == STATICCALL {
		// We do an AddBalance of zero here, just in order to trigger a touch.
		// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
		// but is the correct thing to do and matters on other networks, in tests, and potential
		// future scenarios
		callee.AddBalance(big.NewInt(0))
	}

	gas_copy := gas
	gas_left = gas
	precompiled := self.precompiles.Get(callee.Address())
	code := callee.GetCode()
	start := time.Now()

	if self.vmConfig.Debug {
		if self.depth == 0 {
			self.vmConfig.Tracer.CaptureStart(self, caller.Address(), callee.Address(), precompiled != nil, false, input, gas_copy, value, code)
			defer func() {
				self.vmConfig.Tracer.CaptureEnd(ret, gas-gas_copy, time.Since(start), err)
			}()
		} else {
			self.vmConfig.Tracer.CaptureEnter(typ, caller.Address(), callee.Address(), precompiled != nil, false, input, gas_copy, value, code)
			defer func() {
				self.vmConfig.Tracer.CaptureExit(ret, gas-gas_copy, time.Since(start), err)
			}()
		}
	}

	frame := CallFrame{caller.Account(), callee, input, gas, value}
	if typ == CALLCODE {
		frame = CallFrame{caller.Account(), caller.Account(), input, gas, value}
	} else if typ == DELEGATECALL {
		parent := caller.(*Contract)
		frame = CallFrame{parent.CallerAccount, caller.Account(), input, gas, parent.Value}
	}

	read_only := false
	if typ == STATICCALL {
		read_only = true
	}

	if precompiled != nil {
		if gas_required := precompiled.RequiredGas(frame, self); gas_required <= gas_left {
			gas_left -= gas_required
			gas_copy = gas_required
			ret, err = precompiled.Run(frame, self)
		} else {
			err = ErrOutOfGas
		}
	} else if len(code) != 0 {
		contract := NewContract(frame, CodeAndHash{code, callee.GetCodeHash()})
		ret, err = self.run(&contract, read_only)
		gas_left = contract.Gas
		gas_copy = contract.Gas
	}
	if err != nil {
		self.state.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted && precompiled == nil {
			gas_left = 0
		}
	}
	return
}

// call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (self *EVM) Call(caller ContractRef, callee StateAccount, input []byte, gas uint64, value *big.Int) (ret []byte, gas_left uint64, err error) {
	return self.call(CALL, caller, callee, input, gas, value)
}

// call_code executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// call_code differs from call in the sense that it executes the given address'
// code with the caller as context.
func (self *EVM) call_code(caller ContractRef, callee StateAccount, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	return self.call(CALLCODE, caller, callee, input, gas, value)
}

// call_delegate executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// call_delegate differs from call_code in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (self *EVM) call_delegate(caller ContractRef, callee StateAccount, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	return self.call(DELEGATECALL, caller, callee, input, gas, nil)
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (self *EVM) call_static(caller ContractRef, callee StateAccount, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	return self.call(STATICCALL, caller, callee, input, gas, big.NewInt(0))
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
	stack := newstack()
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
		pc      = uint64(0) // program counter
		cost    uint64
		pcCopy  uint64 // needed for the deferred Tracer
		gasCopy uint64 // for Tracer to log gas remaining before execution
		res     []byte // result of the opcode execution function
	)
	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	if self.vmConfig.Debug {
		defer func() {
			if err != nil {
				self.vmConfig.Tracer.CaptureState(self, pcCopy, op, gasCopy, cost, &mem, stack, contract, self.depth, err)
			}
		}()
	}

	for {
		if self.vmConfig.Debug {
			// Capture pre-execution values for tracing.
			pcCopy, gasCopy = pc, contract.Gas
		}
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := self.instruction_set[op]
		if operation == nil {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}
		if err = operation.validateStack(stack); err != nil {
			return nil, err
		}
		// If the operation is valid, enforce and write restrictions
		if self.read_only {
			// If the interpreter is operating in readonly mode, make sure no
			// state-modifying operation is performed. The 3rd stack item
			// for a call operation is the value. Transferring value from one
			// account to the others means the state is modified and should also
			// return with an error.
			if operation.writes || (op == CALL && stack.Back(2).BitLen() > 0) {
				return nil, ErrWriteProtection
			}
		}
		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		if operation.memorySize != nil {
			memSize, overflow := operation.memorySize(stack).Uint64WithOverflow()
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

		if self.vmConfig.Debug {
			self.vmConfig.Tracer.CaptureState(self, pc, op, gasCopy, cost, &mem, stack, contract, self.depth, err)
		}

		// execute the operation
		res, err = operation.execute(&pc, self, contract, &mem, stack)

		// copy result from the memory
		resCopy := make([]byte, len(res))
		copy(resCopy, res)

		// if the operation clears the return data (e.g. it has returning data)
		// set the last return to the result of the operation.
		if operation.returns {
			self.last_retval = resCopy
		}

		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return resCopy, ErrExecutionReverted
		case operation.halts:
			return resCopy, nil
		case !operation.jumps:
			pc++
		}
	}
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

func (self *EVM) get_account(addr_as_u256 *uint256.Int) StateAccount {
	address := common.Address(addr_as_u256.Bytes20())
	return self.state.GetAccount(&address)
}
