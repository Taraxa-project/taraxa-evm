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
	"golang.org/x/exp/slices"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

// This package implements the main SLASHING contract
// Fixed contract address
var slashing_contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000EE"))

func ContractAddress() *common.Address {
	return slashing_contract_address
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
	ErrNotAValidator               = util.ErrorString("Votes validators is not a validator")
	ErrInvalidVotesPeriodRoundStep = util.ErrorString("Votes period/round/step differ")
	ErrInvalidVotesBlockHash       = util.ErrorString("Invalid votes block hash")
	ErrIdenticalVotes              = util.ErrorString("Votes are identical")
	ErrExistingDoubleVotingProof   = util.ErrorString("Existing double voting proof")
)

// Contract storage fields keys
var (
	field_validators_jail_block = []byte{0}
	field_double_voting_proofs  = []byte{1}
	field_jailed_validators     = []byte{2}
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

func (vote *Vote) GetHash() *common.Hash {
	// Only blockhash and vrf are used for hash, which is signed
	rlp := rlp.MustEncodeToBytes(VoteHashData{BlockHash: vote.BlockHash, VrfSortitionBytes: vote.VrfSortitionBytes})
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
	cfg chain_config.ChainConfig

	// current storage
	storage contract_storage.StorageWrapper
	// delayed storage for PBFT
	delayedReader Reader

	// ABI of the contract
	Abi abi.ABI
	evm *vm.EVM

	logs Logs

	nextCleanUpBlock types.BlockNum
}

// Initialize contract class
func (c *Contract) Init(cfg chain_config.ChainConfig, storage contract_storage.Storage, read_storage Reader, evm *vm.EVM) *Contract {
	c.cfg = cfg
	c.storage.Init(slashing_contract_address, storage)
	c.delayedReader = read_storage
	c.Abi, _ = abi.JSON(strings.NewReader(slashing_sol.TaraxaSlashingClientMetaData))
	c.logs = *new(Logs).Init(c.Abi.Events)
	c.evm = evm
	return c
}

func (c *Contract) storageInitialization() {
	// This needs to be done just once
	if c.storage.GetNonce(slashing_contract_address).Cmp(big.NewInt(0)) == 0 {
		c.storage.IncrementNonce(slashing_contract_address)
	}
}

// Register this precompiled contract
func (c *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *slashing_contract_address
	registry(&defensive_copy, c)
}

// Calculate required gas for call to this contract
func (c *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	method, err := c.Abi.MethodById(ctx.Input)
	if err != nil {
		return 0
	}

	switch method.Name {
	case "commitDoubleVotingProof":
		return CommitDoubleVotingProofGas
	case "getJailBlock":
		return getJailBlockGas
	case "getJailedValidators":
		return getJailBlockGas
	default:
	}

	return DefaultSlashingMethodGas
}

// Should be called on each block commit
func (c *Contract) CommitCall(read_storage Reader) {
	defer c.storage.ClearCache()
	// Update read storage
	c.delayedReader = read_storage
}

// This is called on each call to contract
// It translates call and tries to execute them
func (c *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	method, err := c.Abi.MethodById(ctx.Input)
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

		return nil, c.commitDoubleVotingProof(ctx, evm.GetBlock().Number, args)

	case "getJailBlock":
		var args slashing_sol.ValidatorArg
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getJailBlock input args: ", err)
			return nil, err
		}

		return method.Outputs.Pack(c.getJailBlock(&args.Validator))

	case "getJailedValidators":
		return method.Outputs.Pack(c.delayedReader.GetJailedValidators())
	default:
	}

	return nil, nil
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (c *Contract) commitDoubleVotingProof(ctx vm.CallFrame, block types.BlockNum, args slashing_sol.CommitDoubleVotingProofArgs) error {
	vote_a := NewVote(args.VoteA)
	vote_a_hash := vote_a.GetHash()

	vote_b := NewVote(args.VoteB)
	vote_b_hash := vote_b.GetHash()

	if bytes.Equal(vote_a_hash.Bytes(), vote_b_hash.Bytes()) {
		return ErrIdenticalVotes
	}

	// Check for existing proof
	proof_db_key := c.genDoubleVotingProofDbKey(vote_a_hash, vote_b_hash)
	if c.doubleVotingProoExists(proof_db_key) {
		return ErrExistingDoubleVotingProof
	}

	// Validate votes period and round
	if vote_a.VrfSortition.Period != vote_b.VrfSortition.Period || vote_a.VrfSortition.Round != vote_b.VrfSortition.Round || vote_a.VrfSortition.Step != vote_b.VrfSortition.Step {
		return ErrInvalidVotesPeriodRoundStep
	}

	// Validate voted blocks hashes
	if bytes.Equal(vote_a.BlockHash.Bytes(), vote_b.BlockHash.Bytes()) {
		return ErrInvalidVotesBlockHash
	}

	// Validators can create 2 votes for each second finishing step - one for nullblockhash and one for some specific block
	if vote_a.VrfSortition.Step >= 5 && vote_a.VrfSortition.Step%2 == 1 {
		vote_a_is_zero_hash := bytes.Equal(vote_a.BlockHash.Bytes(), common.ZeroHash.Bytes())
		vote_b_is_zero_hash := bytes.Equal(vote_b.BlockHash.Bytes(), common.ZeroHash.Bytes())

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

	if !bytes.Equal(vote_a_validator.Bytes(), vote_b_validator.Bytes()) {
		return ErrInvalidVotesValidator
	}

	// Check if validator is validator
	if !c.delayedReader.IsValidator(vote_a_validator) {
		return ErrNotAValidator
	}

	// Save jail block for the malicious validator
	jail_block := c.jailValidator(block, vote_a_validator)
	// Save double voting proof
	c.saveDoubleVotingProof(proof_db_key)

	c.evm.AddLog(c.logs.MakeJailedLog(vote_a_validator, block, jail_block, DOUBLE_VOTING))

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
func (c *Contract) jailValidator(current_block types.BlockNum, validator *common.Address) types.BlockNum {
	var jailTime uint64
	if c.cfg.Hardforks.IsOnCactiHardfork(current_block) {
		jailTime = c.cfg.Hardforks.CactiHf.JailTime
	} else {
		jailTime = c.cfg.Hardforks.MagnoliaHf.JailTime
	}

	jail_block := current_block + jailTime
	var current_jail_block *types.BlockNum
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, validator.Bytes())
	c.storage.Get(db_key, func(bytes []byte) {
		current_jail_block = new(types.BlockNum)
		rlp.MustDecodeBytes(bytes, current_jail_block)
	})

	c.storage.Put(db_key, rlp.MustEncodeToBytes(jail_block))
	c.addToJailedValidators(validator)
	// This will be run just once after first write
	c.storageInitialization()

	return jail_block
}

func (c *Contract) addToJailedValidators(validator *common.Address) {
	jailed_validators_key := common.BytesToHash(field_jailed_validators)
	jailed_validators := c.delayedReader.GetJailedValidators()
	if slices.Contains(jailed_validators, *validator) {
		return
	}
	jailed_validators = append(jailed_validators, *validator)
	c.storage.Put(&jailed_validators_key, rlp.MustEncodeToBytes(jailed_validators))
}

func (c *Contract) CleanupJailedValidators(currentBlock types.BlockNum) {
	if c.nextCleanUpBlock > currentBlock {
		return
	}
	// we need it to read current data, not delayed one
	reader := new(Reader).Init(&c.cfg, currentBlock, nil, func(uint64) contract_storage.StorageReader {
		return c.storage
	})
	jailed_validators := reader.GetJailedValidators()

	if len(jailed_validators) == 0 {
		return
	}
	min_unjail_block := uint64(0)
	i := 0
	for _, validator := range jailed_validators {
		_, jail_block := reader.getJailBlock(&validator)

		// keep it in list, if it is not unjailed yet
		if jail_block > currentBlock {
			// copy and increment index
			jailed_validators[i] = validator
			i++
		}

		if min_unjail_block == 0 || jail_block < min_unjail_block {
			min_unjail_block = jail_block
		}
	}
	if min_unjail_block != 0 {
		c.nextCleanUpBlock = min_unjail_block
	}
	// resize list
	jailed_validators = jailed_validators[:i]
	jailed_validators_key := common.BytesToHash(field_jailed_validators)
	c.storage.Put(&jailed_validators_key, rlp.MustEncodeToBytes(jailed_validators))
}

// Return validator's jail time - block until he is jailed. 0 in case he was never jailed
func (c *Contract) getJailBlock(validator *common.Address) types.BlockNum {
	_, jail_block := c.delayedReader.getJailBlock(validator)
	return jail_block
}

func (c *Contract) genDoubleVotingProofDbKey(votea_hash *common.Hash, vote_b_hash *common.Hash) (db_key *common.Hash) {
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

func (c *Contract) saveDoubleVotingProof(db_key *common.Hash) {
	c.storage.Put(db_key, rlp.MustEncodeToBytes(true))
}

func (c *Contract) doubleVotingProoExists(db_key *common.Hash) (ret bool) {
	c.storage.Get(db_key, func(bytes []byte) {
		ret = true
	})

	return
}
