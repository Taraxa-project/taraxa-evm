package slashing

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
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
	getJailBlockGas            uint64 = 5000
	DefaultSlashingMethodGas   uint64 = 5000
)

// Contract methods error return values
var (
	ErrInvalidVoteSignature        = util.ErrorString("Invalid vote signature")
	ErrInvalidVotesValidator       = util.ErrorString("Votes validators differ")
	ErrInvalidVotesPeriodRoundStep = util.ErrorString("Votes period/round/step differ")
	ErrInvalidVotesBlockHash       = util.ErrorString("Invalid votes block hash")
	ErrIdenticalVotes              = util.ErrorString("Votes are identical")
	ErrExistingDoubleVotingProof   = util.ErrorString("Existing double voting proof")
)

// Contract storage fields keys
var (
	field_validators_jail_block = []byte{0}
	field_double_voting_proofs  = []byte{1}
)

type VrfPbftSortition struct {
	Period uint64
	Round  uint32
	Step   uint32
	Proof  [80]byte
}

// Golang representation of C++ vote structure used in consensus
type Vote struct {
	// Block hash
	BlockHash common.Hash

	// Vrf sortition - byte array is used because of the way vote rlp is passed from C++
	VrfSortitionBytes []byte
	VrfSortition      VrfPbftSortition `rlp:"-"`

	// Signature
	Signature [65]byte
}

type VoteHashData struct {
	// Block hash
	BlockHash common.Hash

	// Vrf sortition - byte array is used because of the way vote rlp is passed from C++
	VrfSortitionBytes []byte
}

func (self *Vote) GetHash() *common.Hash {
	// Only blockhash and vrf are used for hash, which is signed
	rlp := rlp.MustEncodeToBytes(VoteHashData{BlockHash: self.BlockHash, VrfSortitionBytes: self.VrfSortitionBytes})
	return keccak256.Hash(rlp)
}

func NewVote(vote_rlp []byte) Vote {
	var vote Vote
	rlp.MustDecodeBytes(vote_rlp, &vote)
	rlp.MustDecodeBytes(vote.VrfSortitionBytes, &vote.VrfSortition)

	return vote
}

// Main contract class
type Contract struct {
	cfg chain_config.MagnoliaHfConfig

	// current storage
	storage contract_storage.StorageWrapper
	// delayed storage for PBFT
	read_storage Reader

	// ABI of the contract
	Abi abi.ABI
	evm *vm.EVM

	logs Logs
}

// Initialize contract class
func (self *Contract) Init(cfg chain_config.MagnoliaHfConfig, storage contract_storage.Storage, read_storage Reader, evm *vm.EVM) *Contract {
	self.cfg = cfg
	self.storage.Init(slashing_contract_address, storage)
	self.read_storage = read_storage
	self.Abi, _ = abi.JSON(strings.NewReader(slashing_sol.TaraxaSlashingClientMetaData))
	self.logs = *new(Logs).Init(self.Abi.Events)
	self.evm = evm
	return self
}

func (self *Contract) storageInitialization() {
	// This needs to be done just once
	if self.storage.GetNonce(slashing_contract_address).Cmp(big.NewInt(0)) == 0 {
		self.storage.IncrementNonce(slashing_contract_address)
	}
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
	case "getJailBlock":
		return getJailBlockGas
	default:
	}

	return DefaultSlashingMethodGas
}

// Should be called on each block commit
func (self *Contract) CommitCall(read_storage Reader) {
	defer self.storage.ClearCache()
	// Update read storage
	self.read_storage = read_storage
}

// This is called on each call to contract
// It translates call and tries to execute them
func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return nil, err
	}

	// First 4 bytes is method signature !!!!
	input := ctx.Input[4:]

	switch method.Name {
	case "commitDoubleVotingProof":
		var args slashing_sol.CommitDoubleVotingProofArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse commitDoubleVotingProof input args: ", err)
			return nil, err
		}

		return nil, self.commitDoubleVotingProof(ctx, evm.GetBlock().Number, args)

	case "getJailBlock":
		var args slashing_sol.ValidatorArg
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getJailBlock input args: ", err)
			return nil, err
		}

		return method.Outputs.Pack(self.getJailBlock(&args.Validator))

	default:
	}

	return nil, nil
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (self *Contract) commitDoubleVotingProof(ctx vm.CallFrame, block types.BlockNum, args slashing_sol.CommitDoubleVotingProofArgs) error {
	vote_a := NewVote(args.VoteA)
	vote_a_hash := vote_a.GetHash()

	vote_b := NewVote(args.VoteB)
	vote_b_hash := vote_b.GetHash()

	if bytes.Compare(vote_a_hash.Bytes(), vote_b_hash.Bytes()) == 0 {
		return ErrIdenticalVotes
	}

	// Check for existing proof
	proof_db_key := self.genDoubleVotingProofDbKey(vote_a_hash, vote_b_hash)
	if self.doubleVotingProoExists(proof_db_key) {
		return ErrExistingDoubleVotingProof
	}

	// Validate votes period and round
	if vote_a.VrfSortition.Period != vote_b.VrfSortition.Period || vote_a.VrfSortition.Round != vote_b.VrfSortition.Round || vote_a.VrfSortition.Step != vote_b.VrfSortition.Step {
		return ErrInvalidVotesPeriodRoundStep
	}

	// Validate voted blocks hashes
	if bytes.Compare(vote_a.BlockHash.Bytes(), vote_b.BlockHash.Bytes()) == 0 {
		return ErrInvalidVotesBlockHash
	}

	// Validators can create 2 votes for each second finishing step - one for nullblockhash and one for some specific block
	if vote_a.VrfSortition.Step >= 5 && vote_a.VrfSortition.Step%2 == 1 {
		vote_a_is_zero_hash := bytes.Compare(vote_a.BlockHash.Bytes(), common.ZeroHash.Bytes()) == 0
		vote_b_is_zero_hash := bytes.Compare(vote_b.BlockHash.Bytes(), common.ZeroHash.Bytes()) == 0

		if (vote_a_is_zero_hash && !vote_b_is_zero_hash) || (!vote_a_is_zero_hash && vote_b_is_zero_hash) {
			return ErrInvalidVotesBlockHash
		}
	}

	vote_a_validator, err := validateVoteSig(vote_a_hash, vote_a.Signature[:])
	if err != nil {
		return ErrInvalidVoteSignature
	}

	vote_b_validator, err := validateVoteSig(vote_b_hash, vote_b.Signature[:])
	if err != nil {
		return ErrInvalidVoteSignature
	}

	if bytes.Compare(vote_a_validator.Bytes(), vote_b_validator.Bytes()) != 0 {
		return ErrInvalidVotesValidator
	}

	// Save jail block for the malicious validator
	jail_block := self.jailValidator(block, vote_a_validator)
	// Save double voting proof
	self.saveDoubleVotingProof(proof_db_key)

	self.evm.AddLog(self.logs.MakeJailedLog(vote_a_validator, block, jail_block, DOUBLE_VOTING))

	return nil
}

func validateVoteSig(vote_hash *common.Hash, signature []byte) (*common.Address, error) {
	// Do not use vote signature to calculate vote hash
	pubKey, err := crypto.Ecrecover(vote_hash.Bytes(), signature)
	if err != nil {
		return nil, err
	}

	return new(common.Address).SetBytes(keccak256.Hash(pubKey[1:])[12:]), nil
}

// Jails validator and returns block number, until which he is jailed
func (self *Contract) jailValidator(current_block types.BlockNum, validator *common.Address) types.BlockNum {
	jail_block := current_block + self.cfg.JailTime

	var currrent_jail_block *types.BlockNum
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, validator.Bytes())
	self.storage.Get(db_key, func(bytes []byte) {
		currrent_jail_block = new(types.BlockNum)
		rlp.MustDecodeBytes(bytes, currrent_jail_block)
	})

	self.storage.Put(db_key, rlp.MustEncodeToBytes(jail_block))

	// This will be run just once after first write
	self.storageInitialization()

	return jail_block
}

// Return validator's jail time - block until he is jailed. 0 in case he was never jailed
func (self *Contract) getJailBlock(validator *common.Address) types.BlockNum {
	_, jail_block := self.read_storage.getJailBlock(validator)
	return jail_block
}

func (self *Contract) genDoubleVotingProofDbKey(votea_hash *common.Hash, vote_b_hash *common.Hash) (db_key *common.Hash) {
	var smaller_vote_hash *common.Hash
	var greater_vote_hash *common.Hash

	// To create the key, hashes must be sorted to have the same key for both combinations of votes
	cmp_res := bytes.Compare(votea_hash.Bytes(), vote_b_hash.Bytes())
	if cmp_res == -1 {
		smaller_vote_hash = votea_hash
		greater_vote_hash = vote_b_hash
	} else if cmp_res == 1 {
		smaller_vote_hash = vote_b_hash
		greater_vote_hash = votea_hash
	} else {
		panic("Votes hashes are the same")
	}

	// Create the key
	db_key = contract_storage.Stor_k_1(field_double_voting_proofs, smaller_vote_hash.Bytes(), greater_vote_hash.Bytes())
	return
}

func (self *Contract) saveDoubleVotingProof(db_key *common.Hash) {
	self.storage.Put(db_key, rlp.MustEncodeToBytes(true))
	return
}

func (self *Contract) doubleVotingProoExists(db_key *common.Hash) (ret bool) {
	ret = false
	self.storage.Get(db_key, func(bytes []byte) {
		ret = true
	})

	return
}
