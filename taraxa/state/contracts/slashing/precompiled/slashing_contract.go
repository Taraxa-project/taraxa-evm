package slashing

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/crypto/secp256k1"
	"github.com/Taraxa-project/taraxa-evm/rlp"
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
	IsJailedGas                uint64 = 5000
	getJailInfoGas             uint64 = 5000
	SlashingGetMethodGas       uint64 = 5000
	DefaultSlashingMethodGas   uint64 = 5000
)

// Contract methods error return values
var (
	ErrInvalidVoteSignature        = util.ErrorString("Invalid vote signature")
	ErrInvalidVotesValidator       = util.ErrorString("Votes validators differs")
	ErrInvalidVotesPeriodRoundStep = util.ErrorString("Votes period/round/step differs")
	ErrInvalidVotesBlockHash       = util.ErrorString("Invalid votes block hash")
	ErrIdenticalVotes              = util.ErrorString("Votes are identical")
	ErrExistingDoubleVotingProof   = util.ErrorString("Existing double voting proof")
)

// Contract storage fields keys
var (
	field_malicious_validators  = []byte{0}
	field_validators_proofs     = []byte{1}
	field_double_voting_proofs  = []byte{2}
	field_validators_jail_block = []byte{3}
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

func (self *Vote) GetRlp(include_sig bool) []byte {
	rlp := rlp.MustEncodeToBytes(self)
	if include_sig {
		return rlp
	}

	return rlp[:len(rlp)-len(self.Signature)]
}

func (self *Vote) GetHash() *common.Hash {
	return keccak256.Hash(self.GetRlp(false))
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
	// delayed storage for PBFT
	read_storage Reader

	// ABI of the contract
	Abi abi.ABI
	evm *vm.EVM

	// Iterable map of malicious validators
	malicious_validators contract_storage.AddressesIMap

	// validator address -> list of proof of his malicious behaviour
	validators_proofs map[common.Address]*ProofsIMap

	// Double voting malicious behaviour proofs
	double_voting_proofs DoubleVotingProofs
}

// Initialize contract class
func (self *Contract) Init(cfg Config, storage contract_storage.Storage, read_storage Reader, evm *vm.EVM) *Contract {
	self.cfg = cfg
	self.storage.Init(slashing_contract_address, storage)
	self.read_storage = read_storage
	self.malicious_validators.Init(&self.storage, field_malicious_validators)
	self.double_voting_proofs.Init(&self.storage, field_double_voting_proofs)
	self.Abi, _ = abi.JSON(strings.NewReader(slashing_sol.TaraxaSlashingClientMetaData))
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
	case "isJailed":
		return IsJailedGas
	case "getJailInfo":
		return getJailInfoGas
	case "getMaliciousValidators":
		validators_count := uint64(self.malicious_validators.GetCount())
		return validators_count * SlashingGetMethodGas
	case "getDoubleVotingProofs":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args slashing_sol.ValidatorArg
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		proofs_count := uint64(self.getValidatorProofsList(&args.Validator).GetCount())
		return proofs_count * SlashingGetMethodGas
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

	case "isJailed":
		var args slashing_sol.ValidatorArg
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse isJailed input args: ", err)
			return nil, err
		}

		return method.Outputs.Pack(self.isJailed(evm.GetBlock().Number, args))

	case "getJailInfo":
		var args slashing_sol.ValidatorArg
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getJailInfo input args: ", err)
			return nil, err
		}

		return method.Outputs.Pack(self.getJailInfo(&args.Validator, true))

	case "getMaliciousValidators":
		return method.Outputs.Pack(self.getMaliciousValidators(evm.GetBlock().Number))

	case "getDoubleVotingProofs":
		var args slashing_sol.ValidatorArg
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getDoubleVotingProofs input args: ", err)
			return nil, err
		}

		return method.Outputs.Pack(self.getDoubleVotingProofs(args))
	default:
	}

	return nil, nil
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (self *Contract) commitDoubleVotingProof(ctx vm.CallFrame, block types.BlockNum, args slashing_sol.CommitDoubleVotingProofArgs) error {
	vote1 := NewVote(args.Vote1)
	vote1_hash := vote1.GetHash()
	vote1_validator, err := validateVoteSig(vote1_hash, vote1.Signature[:])
	if err != nil {
		return ErrInvalidVoteSignature
	}

	vote2 := NewVote(args.Vote2)
	vote2_hash := vote2.GetHash()
	vote2_validator, err := validateVoteSig(vote2_hash, vote2.Signature[:])
	if err != nil {
		return ErrInvalidVoteSignature
	}

	if bytes.Compare(vote1_hash.Bytes(), vote2_hash.Bytes()) == 0 {
		return ErrIdenticalVotes
	}

	if bytes.Compare(vote1_validator.Bytes(), vote2_validator.Bytes()) != 0 {
		return ErrInvalidVotesValidator
	}

	// Check for existing proof
	proof_db_key := self.double_voting_proofs.GenDoubleVotingProofDbKey(vote1_validator, vote1_hash, vote2_hash)
	if self.double_voting_proofs.ProofExists(proof_db_key) {
		return ErrExistingDoubleVotingProof
	}

	// Validate votes period and round
	if vote1.VrfSortition.Period != vote2.VrfSortition.Period || vote1.VrfSortition.Round != vote2.VrfSortition.Round || vote1.VrfSortition.Step != vote2.VrfSortition.Step {
		return ErrInvalidVotesPeriodRoundStep
	}

	// Validate voted blocks hashes
	if bytes.Compare(vote1.BlockHash.Bytes(), vote2.BlockHash.Bytes()) == 0 {
		return ErrInvalidVotesBlockHash
	}

	// Validators can create 2 votes for each second finishing step - one for nullblockhash and one for some specific block
	if vote1.VrfSortition.Step >= 5 && vote1.VrfSortition.Step%2 == 1 {
		vote1_is_zero_hash := bytes.Compare(vote1.BlockHash.Bytes(), common.ZeroHash.Bytes()) == 0
		vote2_is_zero_hash := bytes.Compare(vote2.BlockHash.Bytes(), common.ZeroHash.Bytes()) == 0

		if (vote1_is_zero_hash && !vote2_is_zero_hash) || (!vote1_is_zero_hash && vote2_is_zero_hash) {
			return ErrInvalidVotesBlockHash
		}
	}

	// Save the proof into db
	proof := slashing_sol.SlashingInterfaceDoubleVotingProof{ProofAuthor: *ctx.CallerAccount.Address(), Block: big.NewInt(int64(block))}
	self.double_voting_proofs.SaveProof(proof_db_key, &proof)

	// Assign proof db key to the specific malicious validator
	validator_proofs := self.getValidatorProofsList(vote1_validator)
	validator_proofs.CreateProof(DoubleVoting, proof_db_key)

	// Add validator to the list of malicious validators
	if !self.malicious_validators.AccountExists(vote1_validator) {
		self.malicious_validators.CreateAccount(vote1_validator)
	}

	// Save jail block for the malicious validator
	self.jailValidator(block, vote1_validator)

	return nil
}

func (self *Contract) getValidatorProofsList(validator *common.Address) *ProofsIMap {
	validator_proofs, found := self.validators_proofs[*validator]
	if found == false {
		validator_proofs = new(ProofsIMap)
		validator_proofs_field := append(field_validators_proofs, validator[:]...)
		validator_proofs.Init(&self.storage, validator_proofs_field)
	}

	return validator_proofs
}

func validateVoteSig(vote_hash *common.Hash, signature []byte) (*common.Address, error) {
	// Do not use vote signature to calculate vote hash
	pubKey, err := secp256k1.RecoverPubkey(vote_hash.Bytes(), signature)
	if err != nil {
		return nil, err
	}

	return new(common.Address).SetBytes(keccak256.Hash(pubKey[1:])[12:]), nil
}

func (self *Contract) jailValidator(current_block types.BlockNum, validator *common.Address) {
	jail_block := current_block + self.cfg.DoubleVotingJailTime

	var currrent_jail_block *types.BlockNum
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, validator.Bytes())
	self.storage.Get(db_key, func(bytes []byte) {
		currrent_jail_block = new(types.BlockNum)
		rlp.MustDecodeBytes(bytes, currrent_jail_block)
	})

	// In case validator is already jailed, compound his jail time
	if currrent_jail_block != nil && *currrent_jail_block+self.cfg.DoubleVotingJailTime > jail_block {
		jail_block = *currrent_jail_block + self.cfg.DoubleVotingJailTime
	}

	self.storage.Put(db_key, rlp.MustEncodeToBytes(jail_block))
	// This will be run just once after first write
	self.storageInitialization()
}

// Return validator's jail time - block until he is jailed. 0 in case he was never jailed
func (self *Contract) getJailInfo(validator *common.Address, get_proofs_count bool) (ret slashing_sol.SlashingInterfaceJailInfo) {
	ret = self.read_storage.getJailInfo(validator)

	if get_proofs_count == false {
		ret.ProofsCount = 0
		return
	}

	ret.ProofsCount = self.getValidatorProofsList(validator).GetCount()
	return
}

func (self *Contract) isJailed(block types.BlockNum, args slashing_sol.ValidatorArg) bool {
	return self.read_storage.IsJailed(block, &args.Validator)
}

func (self *Contract) getMaliciousValidators(block types.BlockNum) (ret []slashing_sol.SlashingInterfaceMaliciousValidator) {
	malicious_validators, _ := self.malicious_validators.GetAccounts(0, self.malicious_validators.GetCount())

	// Reserve slice capacity
	ret = make([]slashing_sol.SlashingInterfaceMaliciousValidator, len(malicious_validators))

	for idx, validator_address := range malicious_validators {
		ret[idx] = slashing_sol.SlashingInterfaceMaliciousValidator{Validator: validator_address, JailInfo: self.getJailInfo(&validator_address, true)}
	}

	return
}

func (self *Contract) getDoubleVotingProofs(args slashing_sol.ValidatorArg) (ret []slashing_sol.SlashingInterfaceDoubleVotingProof) {
	validator_proofs_list := self.getValidatorProofsList(&args.Validator)

	proofs_keys, _ := validator_proofs_list.GetProofs(0, validator_proofs_list.GetCount())

	// Reserve slice capacity
	ret = make([]slashing_sol.SlashingInterfaceDoubleVotingProof, 0)

	for _, proof_key := range proofs_keys {
		if proof_key.Proof_type != DoubleVoting {
			continue
		}

		proof := self.double_voting_proofs.GetProof(&proof_key.DbKey)
		ret = append(ret, *proof)
	}

	return
}
