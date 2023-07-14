package slashing

import (
	"fmt"

	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

// This package implements the main SLASHING contract
// Fixed contract address
var slashing_contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000EE"))

func ContractAddress() common.Address {
	return *slashing_contract_address
}

// Gas constants - gas is determined based on storage writes. Each 32Bytes == 20k gas
const (
	CommitDoubleVotingProofGas uint64 = 20000
	IsJailedGas                uint64 = 5000
	DefaultSlashingMethodGas   uint64 = 5000
)

// Contract methods error return values
var (
	ErrInsufficientBalance       = util.ErrorString("Insufficient balance")
	ErrCallIsNotToplevel         = util.ErrorString("only top-level calls are allowed")
	ErrWrongDoubleVotingProof    = util.ErrorString("Wrong double voting proof, validator address could not be recovered")
	ErrExistingDoubleVotingProof = util.ErrorString("Existing double voting proof")
)

// Contract storage fields keys
var (
	field_jail_block = []byte{0}
)

// Main contract class
type Contract struct {
	cfg Config
	// current storage
	storage contract_storage.StorageWrapper
	// ABI of the contract
	Abi abi.ABI
	evm *vm.EVM
}

// Initialize contract class
func (self *Contract) Init(cfg Config, storage contract_storage.Storage, readStorage Reader, evm *vm.EVM) *Contract {
	self.cfg = cfg
	self.storage.Init(slashing_contract_address, storage)
	self.evm = evm
	return self
}

// Updates config - for HF
func (self *Contract) UpdateConfig(cfg Config) {
	self.cfg = cfg
}

// Register this precompiled contract
func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *slashing_contract_address
	registry(&defensive_copy, self)
}

// Calculate required gas for call to this contract
func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return 0
	}

	switch method.Name {
	case "commitDoubleVotingProof":
		return CommitDoubleVotingProofGas
	case "isJailed":
		return IsJailedGas
	default:
	}

	return DefaultSlashingMethodGas
}

// Should be called on each block commit - updates delayedStorage
func (self *Contract) CommitCall(readStorage Reader) {
	defer self.storage.ClearCache()
}

// This is called on each call to contract
// It translates call and tries to execute them
func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}

	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return nil, err
	}

	// First 4 bytes is method signature !!!!
	input := ctx.Input[4:]

	switch method.Name {
	case "commitDoubleVotingProof":
		var args sol.CommitDoubleVotingProofArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse commitDoubleVotingProof input args: ", err)
			return nil, err
		}

		return nil, self.commitDoubleVotingProof(ctx, evm.GetBlock().Number, args)
	case "isJailed":
		var args sol.IsJailedArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse IsJailed input args: ", err)
			return nil, err
		}

		// TODO: fix
		//return method.Outputs.Pack(self.storage.IsJailed(&args.Validator))
		return method.Outputs.Pack(false)
	default:
	}

	return nil, nil
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (self *Contract) commitDoubleVotingProof(ctx vm.CallFrame, block types.BlockNum, args sol.CommitDoubleVotingProofArgs) error {

	return nil
}

// func validateProof(proof []byte, validator *common.Address) error {
// 	if len(proof) != 65 {
// 		return ErrWrongProof
// 	}

// 	// Make sure the public key is a valid one
// 	pubKey, err := crypto.Ecrecover(keccak256.Hash(validator.Bytes()).Bytes(), append(proof[:64], proof[64]-27))
// 	if err != nil {
// 		return err
// 	}

// 	// the first byte of pubkey is bitcoin heritage
// 	if common.BytesToAddress(keccak256.Hash(pubKey[1:])[12:]) != *validator {
// 		return ErrWrongProof
// 	}

// 	return nil
// }

// Returns batch of delegations for specified delegator address
func (self *Contract) isJailed(args sol.IsJailedArgs) bool {
	// delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(&args.Delegator, args.Batch, GetDelegationsMaxCount)

	return false
}

// func (self *Contract) state_get(validator_addr, block []byte) (state *State, key common.Hash) {
// 	key = stor_k_2(field_state, validator_addr, block)
// 	self.storage.Get(&key, func(bytes []byte) {
// 		state = new(State)
// 		rlp.MustDecodeBytes(bytes, state)
// 	})
// 	return
// }

// // Saves state object to storage
// func (self *Contract) state_put(key *common.Hash, state *State) {
// 	if state != nil {
// 		self.storage.Put(key, rlp.MustEncodeToBytes(state))
// 	} else {
// 		self.storage.Put(key, nil)
// 	}
// }
