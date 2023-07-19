package slashing

import (
	"fmt"
	"log"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/crypto/secp256k1"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	slashing_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"

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
	ErrInsufficientBalance         = util.ErrorString("Insufficient balance")
	ErrCallIsNotToplevel           = util.ErrorString("only top-level calls are allowed")
	ErrInvalidVoteSignature        = util.ErrorString("Invalid vote signature")
	ErrInvalidVotesValidator       = util.ErrorString("Votes validator differs")
	ErrInvalidVotesPeriodRoundStep = util.ErrorString("Votes period/round/step differs")
	ErrInvalidDoubleVotingProof    = util.ErrorString("Wrong double voting proof, validator address could not be recovered")
	ErrExistingDoubleVotingProof   = util.ErrorString("Existing double voting proof")
)

// Contract storage fields keys
var (
	field_jail_block = []byte{0}
)

type VrfPbftSortition struct {
	Period uint64
	Round  uint32
	Step   uint32
	Proof  [80]byte
}

// Gong representation of C++ vote structure used in consensus
type Vote struct {
	// Block hash
	BlockHash common.Hash

	// Vrf sortition
	VrfSortition VrfPbftSortition

	// Signature
	Signature [65]byte
}

func (self *Vote) GetVoteRlp(include_sig bool) []byte {
	rlp := rlp.MustEncodeToBytes(self)
	if include_sig {
		return rlp
	}

	return rlp[:len(rlp)-len(self.Signature)]
}

func NewVote(vote_rlp []byte) Vote {
	var vote Vote
	rlp.MustDecodeBytes(vote_rlp, &vote)

	return vote
}

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
	self.Abi, _ = abi.JSON(strings.NewReader(slashing_sol.TaraxaSlashingClientMetaData))
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

		return method.Outputs.Pack(self.isJailed(args))
	default:
	}

	return nil, nil
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (self *Contract) commitDoubleVotingProof(ctx vm.CallFrame, block types.BlockNum, args sol.CommitDoubleVotingProofArgs) error {
	vote1 := NewVote(args.Vote1)
	vote1_validator, err := validateVoteSig(vote1)
	if err != nil {
		return ErrInvalidVoteSignature
	}

	vote2 := NewVote(args.Vote2)
	vote2_validator, err := validateVoteSig(vote2)
	if err != nil {
		return ErrInvalidVoteSignature
	}

	// Check if votes validator is the same address
	if *vote1_validator != *vote2_validator {
		return ErrInvalidDoubleVotingProof
	}

	// Validate votes period and round
	if vote1.VrfSortition.Period != vote2.VrfSortition.Period || vote1.VrfSortition.Round != vote2.VrfSortition.Round {
		return ErrInvalidVotesPeriodRoundStep
	}

	// TODO: Validates votes step

	// TODO: save proof into db...

	log.Println("vote1: ", vote1)
	log.Println("vote2: ", vote2)

	return nil
}

func validateVoteSig(vote Vote) (*common.Address, error) {
	// Do not use vote signature to calculate vote hash
	pubKey, err := secp256k1.RecoverPubkey(keccak256.Hash(vote.GetVoteRlp(false)).Bytes(), vote.Signature[:])
	if err != nil {
		return nil, err
	}

	return new(common.Address).SetBytes(keccak256.Hash(pubKey[1:])[12:]), nil
}

// Returns batch of delegations for specified delegator address
func (self *Contract) isJailed(args sol.IsJailedArgs) bool {
	return false
}
