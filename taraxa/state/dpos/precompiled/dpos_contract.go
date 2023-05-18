package dpos

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/solidity"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
)

// This package implements the main DPOS contract as well as the fee distribution schema
// Fee distribution is based on the "F1 Fee Distribution" algorithm
////////////////////////////////////////////////////////////////////////////
// The key point of how F1 works is that it tracks how much rewards a delegator with 1
// stake delegated to a given validator would be entitled to if it had bonded at block 0 until
// the latest block. When a delegator bonds at block b, the amount of rewards a delegator
// with 1 stake would have if bonded at block 0 until block b is also persisted to state. When
// the delegator withdraws, they receive the difference of these two values. Since rewards
// are distributed according to stake-weighting, this amount of rewards can be scaled by the
// amount of stake a delegator had delegated. [1]
////////////////////////////////////////////////////////////////////////////
// [1] https://drops.dagstuhl.de/opus/volltexte/2020/11974/pdf/OASIcs-Tokenomics-2019-10.pdf

// Fixed contract address
var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000FE"))

// Gas constants - gas is determined based on storage writes. Each 32Bytes == 20k gas
const (
	RegisterValidatorGas      uint64 = 80000
	SetCommissionGas          uint64 = 20000
	DelegateGas               uint64 = 40000
	UndelegateGas             uint64 = 60000
	ConfirmUndelegateGas      uint64 = 20000
	CancelUndelegateGas       uint64 = 60000
	ReDelegateGas             uint64 = 80000
	ClaimRewardsGas           uint64 = 40000
	ClaimCommissionRewardsGas uint64 = 20000
	SetValidatorInfoGas       uint64 = 20000
	DposGetMethodsGas         uint64 = 5000
	DposBatchGetMethodsGas    uint64 = 5000
	DefaultDposMethodGas      uint64 = 20000
)

// Contract methods error return values
var (
	ErrInsufficientBalance          = util.ErrorString("Insufficient balance")
	ErrNonExistentValidator         = util.ErrorString("Validator does not exist")
	ErrNonExistentDelegation        = util.ErrorString("Delegation does not exist")
	ErrExistentDelegation           = util.ErrorString("Delegation already exist")
	ErrExistentUndelegation         = util.ErrorString("Undelegation already exist")
	ErrNonExistentUndelegation      = util.ErrorString("Undelegation does not exist")
	ErrLockedUndelegation           = util.ErrorString("Undelegation is not yet ready to be withdrawn")
	ErrExistentValidator            = util.ErrorString("Validator already exist")
	ErrBrokenState                  = util.ErrorString("Fatal error state is broken")
	ErrValidatorsMaxStakeExceeded   = util.ErrorString("Validator's max stake exceeded")
	ErrInsufficientDelegation       = util.ErrorString("Insufficient delegation")
	ErrCallIsNotToplevel            = util.ErrorString("only top-level calls are allowed")
	ErrWrongProof                   = util.ErrorString("Wrong proof, validator address could not be recovered")
	ErrWrongOwnerAcc                = util.ErrorString("This account is not owner of specified validator")
	ErrWrongVrfKey                  = util.ErrorString("Wrong vrf key specified in validator arguments")
	ErrForbiddenCommissionChange    = util.ErrorString("Forbidden commission change")
	ErrCommissionOverflow           = util.ErrorString("Commission is bigger than maximum value")
	ErrMaxEndpointLengthExceeded    = util.ErrorString("Max endpoint length exceeded")
	ErrMaxDescriptionLengthExceeded = util.ErrorString("Max description length exceeded")
)

const (
	// Max num of characters in url
	MaxEndpointLength = 50

	// Max num of characters in description
	MaxDescriptionLength = 100

	// Maximal commission  [%] * 100 so 1% is 100 & 100% is 10000
	MaxCommission = uint64(10000)

	// Length of vrf public key
	VrfKeyLength = 32

	// Maximum number of validators per batch that delegator get claim rewards from
	ClaimAllRewardsMaxCount = 10

	// Maximum number of validators per batch returned by getValidators call
	GetValidatorsMaxCount = 20

	// Maximum number of validators per batch returned by getDelegations call
	GetDelegationsMaxCount = 20

	// Maximum number of validators per batch returned by getUndelegations call
	GetUndelegationsMaxCount = 20
)

// Contract storage fields keys
var (
	field_validators    = []byte{0}
	field_state         = []byte{1}
	field_delegations   = []byte{2}
	field_undelegations = []byte{3}

	field_eligible_vote_count = []byte{4}
	field_amount_delegated    = []byte{5}
)

// State of the rewards distribution algorithm
type State struct {
	// represents number of rewards per 1 stake
	RewardsPer1Stake *big.Int
	// number of references
	Count uint32
}

// Main contract class
type Contract struct {
	// Configs
	dpos_config      chain_config.DposConfig
	hardforks_config chain_config.HardforksConfig

	// current storage
	storage StorageWrapper
	// delayed storage for PBFT
	delayedStorage Reader
	// ABI of the contract
	Abi  abi.ABI
	logs Logs
	evm  *vm.EVM

	// Iterable storages
	validators    Validators
	delegations   Delegations
	undelegations Undelegations

	// values for PBFT
	eligible_vote_count_orig uint64
	eligible_vote_count      uint64
	amount_delegated_orig    *uint256.Int
	amount_delegated         *uint256.Int
	blocks_per_year          *uint256.Int
	yield_percentage         *uint256.Int
	dag_proposers_reward     *uint256.Int
	max_block_author_reward  *uint256.Int

	lazy_init_done bool
}

// Initialize contract class
func (self *Contract) Init(dpos_config chain_config.DposConfig, hardforks_config chain_config.HardforksConfig, storage Storage, readStorage Reader, evm *vm.EVM) *Contract {
	self.dpos_config = dpos_config
	self.hardforks_config = hardforks_config
	self.storage.Init(storage)
	self.delayedStorage = readStorage
	self.evm = evm
	return self
}

// Updates delayed storage after each commited block
func (self *Contract) UpdateStorage(readStorage Reader) {
	self.delayedStorage = readStorage
}

// Updates config - for HF
func (self *Contract) UpdateConfig(dpos_config chain_config.DposConfig) {
	self.dpos_config = dpos_config
}

// Register this precompiled contract
func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

// Calculate required gas for call to this contract
func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	// Init abi and some of the structures required for calculating gas, e.g. self.validators for getValidators
	self.lazy_init()

	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return 0
	}

	switch method.Name {
	case "delegate":
		return DelegateGas
	case "undelegate":
		return UndelegateGas
	case "confirmUndelegate":
		return ConfirmUndelegateGas
	case "cancelUndelegate":
		return CancelUndelegateGas
	case "reDelegate":
		return ReDelegateGas
	case "claimCommissionRewards":
		return ClaimCommissionRewardsGas
	case "setCommission":
		return SetCommissionGas
	case "registerValidator":
		return RegisterValidatorGas
	case "setValidatorInfo":
		return SetValidatorInfoGas
	case "isValidatorEligible":
	case "getTotalEligibleVotesCount":
	case "getValidatorEligibleVotesCount":
	case "getValidator":
		return DposGetMethodsGas
	case "claimRewards":
		return ClaimRewardsGas
	case "claimAllRewards":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args sol.ClaimAllRewardsArgs
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		delegations_count := self.batch_items_count(uint64(self.delegations.GetDelegationsCount(ctx.CallerAccount.Address())), uint64(args.Batch), ClaimAllRewardsMaxCount)
		// delegations_count * DposBatchGetMethodsGas is the price for getting all validators from db(1:1 to getValidators gas) and
		// delegations_count * ClaimRewardsGas is for calling claimRewards for each validator
		return delegations_count * (DposBatchGetMethodsGas + ClaimRewardsGas)
	case "getValidators":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args sol.GetValidatorsArgs
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		validators_count := self.batch_items_count(uint64(self.validators.GetValidatorsCount()), uint64(args.Batch), GetValidatorsMaxCount)
		return validators_count * DposBatchGetMethodsGas

	case "getValidatorsFor":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args sol.GetValidatorsForArgs
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		// This method is iterating through list of validators, so we charge relatively large fee here
		return GetValidatorsMaxCount * DposBatchGetMethodsGas

	case "getDelegations":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args sol.GetDelegationsArgs
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		delegations_count := self.batch_items_count(uint64(self.delegations.GetDelegationsCount(&args.Delegator)), uint64(args.Batch), GetDelegationsMaxCount)
		return delegations_count * DposBatchGetMethodsGas

	case "getUndelegations":
		// First 4 bytes is method signature !!!!
		input := ctx.Input[4:]
		var args sol.GetUndelegationsArgs
		if err := method.Inputs.Unpack(&args, input); err != nil {
			// args parsing will fail also during Run() so the tx wont get executed
			return 0
		}

		undelegations_count := self.batch_items_count(uint64(self.undelegations.GetUndelegationsCount(&args.Delegator)), uint64(args.Batch), GetUndelegationsMaxCount)
		return undelegations_count * DposBatchGetMethodsGas
	default:
	}

	return DefaultDposMethodGas
}

func (self *Contract) batch_items_count(actual_count uint64, batch uint64, max_batch_items_count uint64) uint64 {
	// In case there are no validators, charge as for standard get method as counter must have been read from db
	if actual_count == 0 {
		return 1
	}

	// Wrong batch specified - there are no more validators for specified batch, charge as for standard get method as counter must have been read from db
	batch_shift_count := batch * max_batch_items_count
	if batch_shift_count >= actual_count {
		return 1
	}

	items_to_be_returned_count := actual_count - batch_shift_count

	// There is a hard cap of max num of returned validators
	if items_to_be_returned_count > max_batch_items_count {
		return max_batch_items_count
	}

	return items_to_be_returned_count
}

// Lazy initialization only if contract is needed
func (self *Contract) lazy_init() {
	if self.lazy_init_done {
		return
	}

	self.Abi, _ = abi.JSON(strings.NewReader(sol.TaraxaDposClientMetaData))
	self.logs = *new(Logs).Init(self.Abi.Events)

	self.validators.Init(&self.storage, field_validators)
	self.delegations.Init(&self.storage, field_delegations)
	self.undelegations.Init(&self.storage, field_undelegations)

	self.blocks_per_year = uint256.NewInt(uint64(self.dpos_config.BlocksPerYear))
	self.yield_percentage = uint256.NewInt(uint64(self.dpos_config.YieldPercentage))

	self.dag_proposers_reward = uint256.NewInt(uint64(self.dpos_config.DagProposersReward))
	self.max_block_author_reward = uint256.NewInt(uint64(self.dpos_config.MaxBlockAuthorReward))

	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		self.eligible_vote_count_orig = bin.DEC_b_endian_compact_64(bytes)
	})
	self.eligible_vote_count = self.eligible_vote_count_orig

	self.amount_delegated_orig = uint256.NewInt(0)
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		self.amount_delegated_orig = new(uint256.Int).SetBytes(bytes)
	})
	self.amount_delegated = self.amount_delegated_orig.Clone()

	self.lazy_init_done = true
}

// Should be called from EndBlock on each block
func (self *Contract) EndBlockCall() {
	if !self.lazy_init_done {
		return
	}
	// Update values
	if self.eligible_vote_count_orig != self.eligible_vote_count {
		self.storage.Put(stor_k_1(field_eligible_vote_count), bin.ENC_b_endian_compact_64_1(self.eligible_vote_count))
		self.eligible_vote_count_orig = self.eligible_vote_count
	}
	if self.amount_delegated_orig.Cmp(self.amount_delegated) != 0 {
		self.storage.Put(stor_k_1(field_amount_delegated), self.amount_delegated.Bytes())
		self.amount_delegated_orig = self.amount_delegated.Clone()
	}
}

// Should be called on each block commit - updates delayedStorage
func (self *Contract) CommitCall(readStorage Reader) {
	defer self.storage.ClearCache()
	// Storage Update
	self.delayedStorage = readStorage
}

// Fills contract based on genesis values
func (self *Contract) ApplyGenesis(get_account func(*common.Address) vm.StateAccount) error {
	self.lazy_init()

	make_context := func(caller *common.Address, value *big.Int) (ctx vm.CallFrame) {
		ctx.CallerAccount = get_account(caller)
		ctx.Account = get_account(contract_address)
		ctx.Value = value
		return
	}

	for _, entry := range self.dpos_config.InitialValidators {
		self.apply_genesis_entry(&entry, make_context)
	}

	self.EndBlockCall()
	self.storage.IncrementNonce(contract_address)

	return nil
}

// This is called on each call to contract
// It translates call and tries to execute them
func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}

	self.lazy_init()

	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return nil, err
	}

	// First 4 bytes is method signature !!!!
	input := ctx.Input[4:]

	switch method.Name {
	case "delegate":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse delegate input args: ", err)
			return nil, err
		}
		return nil, self.delegate(ctx, evm.GetBlock().Number, args)

	case "undelegate":
		var args sol.UndelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse delegate input args: ", err)
			return nil, err
		}
		return nil, self.undelegate(ctx, evm.GetBlock().Number, args)

	case "confirmUndelegate":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse confirmUndelegate input args: ", err)
			return nil, err
		}
		return nil, self.confirmUndelegate(ctx, evm.GetBlock().Number, args)

	case "cancelUndelegate":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse cancelUndelegate input args: ", err)
			return nil, err
		}

		return nil, self.cancelUndelegate(ctx, evm.GetBlock().Number, args)

	case "reDelegate":
		var args sol.RedelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse reDelegate input args: ", err)
			return nil, err
		}
		return nil, self.redelegate(ctx, evm.GetBlock().Number, args)

	case "claimRewards":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimRewards input args: ", err)
			return nil, err
		}
		return nil, self.claimRewards(ctx, evm.GetBlock().Number, args)

	case "claimAllRewards":
		var args sol.ClaimAllRewardsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimAllRewards input args: ", err)
			return nil, err
		}

		result, err := self.claimAllRewards(ctx, evm.GetBlock().Number, args)
		if err != nil {
			return nil, err
		}
		return method.Outputs.Pack(result)

	case "claimCommissionRewards":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimCommissionRewards input args: ", err)
			return nil, err
		}
		return nil, self.claimCommissionRewards(ctx, evm.GetBlock().Number, args)

	case "setCommission":
		var args sol.SetCommissionArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimCommissionRewards input args: ", err)
			return nil, err
		}
		return nil, self.setCommission(ctx, evm.GetBlock().Number, args)

	case "registerValidator":
		var args sol.RegisterValidatorArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse registerValidator input args: ", err)
			return nil, err
		}
		return nil, self.registerValidator(ctx, evm.GetBlock().Number, args)

	case "setValidatorInfo":
		var args sol.SetValidatorInfoArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse setValidatorInfo input args: ", err)
			return nil, err
		}
		return nil, self.setValidatorInfo(ctx, args)

	case "isValidatorEligible":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse isValidatorEligible input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.delayedStorage.IsEligible(&args.Validator))

	case "getTotalEligibleVotesCount":
		return method.Outputs.Pack(self.delayedStorage.EligibleVoteCount())

	case "getValidatorEligibleVotesCount":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.delayedStorage.GetEligibleVoteCount(&args.Validator))

	case "getValidator":
		var args sol.ValidatorAddressArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidator input args: ", err)
			return nil, err
		}
		result, err := self.getValidatorInfo(evm.GetBlock().Number, args)
		if err != nil {
			return nil, err
		}
		return method.Outputs.Pack(result)

	case "getValidators":
		var args sol.GetValidatorsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidators input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.getValidators(evm.GetBlock().Number, args))

	case "getValidatorsFor":
		var args sol.GetValidatorsForArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorsFor input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.getValidatorsFor(evm.GetBlock().Number, args))

	case "getDelegations":
		var args sol.GetDelegationsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getDelegations input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.getDelegations(evm.GetBlock().Number, args))

	case "getUndelegations":
		var args sol.GetUndelegationsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getUndelegations input args: ", err)
			return nil, err
		}
		return method.Outputs.Pack(self.getUndelegations(args))
	default:
	}

	return nil, nil
}

// ----------------------------------------------------------------
// Brief description of distribution algorithm
// ----------------------------------------------------------------
// - Total block reward - `blockReward` is calculated  based on yield_percentage
// - Block reward is distributed based on `VotesToTransactionsRatio` between votes and transactions
// - Then bonus reward is calculated based on MaxBlockAuthorReward
// - Vote reward is reduced by bonus reward
// - Bonus reward is theoretical and it will be added to block proposer (author) only when all votes are included
// - If less reward votes are included, rest of the bonus reward it is just burned
// - Then for each validator vote and transaction proportion rewards are calculated and distributed

func (self *Contract) DistributeRewards(block types.BlockNum, rewardsStats *rewards_stats.RewardsStats, feesRewards *FeesRewards) *uint256.Int {
	// When calling DistributeRewards, internal structures must be always initialized
	self.lazy_init()
	blockAuthorAddr := &rewardsStats.BlockAuthor

	// Calculates number of tokens to be generated as block reward
	blockReward := new(uint256.Int).Mul(self.amount_delegated, self.yield_percentage)
	blockReward.Div(blockReward, new(uint256.Int).Mul(uint256.NewInt(100), self.blocks_per_year))

	totalReward := uint256.NewInt(0)
	votesReward := uint256.NewInt(0)
	blockAuthorReward := uint256.NewInt(0)
	dagProposersReward := blockReward.Clone()
	// We need to handle case for block 1
	if rewardsStats.TotalVotesWeight > 0 {
		// Calculate propotion between votes and transactions
		dagProposersReward.Div(new(uint256.Int).Mul(blockReward, self.dag_proposers_reward), uint256.NewInt(100))
		votesReward.Sub(blockReward, dagProposersReward)

		// Calculate bonus reward as part of blockReward multiplied by MaxBlockAuthorReward and subtract it from total votes reward part. As the reward part of Dag defined above and it should not change
		bonusReward := new(uint256.Int).Div(new(uint256.Int).Mul(blockReward, self.max_block_author_reward), uint256.NewInt(100))
		votesReward.Sub(votesReward, bonusReward)

		// As MaxVotesWeight is just theoretical value we need to have use max of those
		maxVotesWeigh := Max(rewardsStats.MaxVotesWeight, rewardsStats.TotalVotesWeight)

		// In case all reward votes are included we will just pass block author whole reward, this should improve rounding issues
		if maxVotesWeigh == rewardsStats.TotalVotesWeight {
			blockAuthorReward = bonusReward
		} else {
			twoTPlusOne := maxVotesWeigh*2/3 + 1
			bonusVotesWeight := uint64(0)
			if rewardsStats.TotalVotesWeight >= twoTPlusOne {
				bonusVotesWeight = rewardsStats.TotalVotesWeight - twoTPlusOne
			} else {
				errorString := fmt.Sprintf("DistributeRewards - TotalVotesWeight (%d) is smaller than two twoTPlusOne (%d)", rewardsStats.TotalVotesWeight, twoTPlusOne)
				fmt.Println(errorString)
			}
			// should be zero if rewardsStats.TotalVotesWeight == twoTPlusOne
			blockAuthorReward.Div(new(uint256.Int).Mul(bonusReward, uint256.NewInt(uint64(bonusVotesWeight))), uint256.NewInt(uint64(maxVotesWeigh-twoTPlusOne)))
		}
	}

	newMintedRewards := uint256.NewInt(0)
	// Add reward to the block author for additional included votes
	if blockAuthorReward.Cmp(uint256.NewInt(0)) == 1 {
		block_author := self.validators.GetValidator(blockAuthorAddr)
		if block_author != nil {
			commission := new(uint256.Int).Div(new(uint256.Int).Mul(blockAuthorReward, uint256.NewInt(uint64(block_author.Commission))), uint256.NewInt(MaxCommission))
			delegatorsRewards := new(uint256.Int).Sub(blockAuthorReward, commission)
			self.validators.AddValidatorRewards(blockAuthorAddr, commission.ToBig(), delegatorsRewards.ToBig())
			newMintedRewards.Add(newMintedRewards, blockAuthorReward)
			totalReward.Add(totalReward, blockAuthorReward)
		}
	}

	TotalDagBlocksCountCheck := uint32(0)
	totalVoteWeightCheck := uint64(0)
	// Calculates validators rewards (for dpos blocks producers, block voters)
	for validatorAddress, validatorStats := range rewardsStats.ValidatorsStats {
		// We need to calculate validator reward even though in some edge cases this validator might not exist in contract anymore
		// If we would not calculate it, totalUniqueTrxsCountCheck, totalVoteWeightCheck and newMintedRewards might not pass
		validatorReward := uint256.NewInt(0)
		// Calculate it like this to eliminate rounding error as much as possible
		// Reward for DAG blocks with at least one unique transaction
		if validatorStats.DagBlocksCount > 0 {
			TotalDagBlocksCountCheck += validatorStats.DagBlocksCount
			validatorReward.Mul(uint256.NewInt(uint64(validatorStats.DagBlocksCount)), dagProposersReward)
			validatorReward.Div(validatorReward, uint256.NewInt(uint64(rewardsStats.TotalDagBlocksCount)))
		}

		// Add reward for voting
		if validatorStats.VoteWeight > 0 {
			totalVoteWeightCheck += validatorStats.VoteWeight
			// total_votes_reward * validator_vote_weight / total_votes_weight
			validatorVoteReward := new(uint256.Int).Mul(uint256.NewInt(uint64(validatorStats.VoteWeight)), votesReward)
			validatorVoteReward.Div(validatorVoteReward, uint256.NewInt(uint64(rewardsStats.TotalVotesWeight)))
			validatorReward.Add(validatorReward, validatorVoteReward)
		}

		// Add reward for for final check
		newMintedRewards.Add(newMintedRewards, validatorReward)

		validator := self.validators.GetValidator(&validatorAddress)
		if validator == nil {
			// This could happen due to few blocks artificial delay we use to determine if validator is eligible or not when
			// checking it during consensus. If everyone undelegates from validator and also he claims his commission rewards
			// during the the period of time, which is < then delay we use, he is deleted from contract storage, but he will be
			// able to propose few more blocks. This situation is extremely unlikely, but technically possible.
			// If it happens, validator will simply not receive rewards for those few last blocks/votes he produced
			// We shouldn't really check if validator was eligible before. Because there is a possibility to include some old DAG block anytime(not only at this few blocks old)
			continue
		}

		// Adds fees for all txs that validator added in his blocks as first
		validatorReward.Add(validatorReward, feesRewards.GetTrxsFeesReward(validatorAddress))
		totalReward.Add(totalReward, validatorReward)

		validatorCommission := new(uint256.Int).Div(new(uint256.Int).Mul(validatorReward, uint256.NewInt(uint64(validator.Commission))), uint256.NewInt(MaxCommission))
		delegatorRewards := new(uint256.Int).Sub(validatorReward, validatorCommission)

		self.validators.AddValidatorRewards(&validatorAddress, validatorCommission.ToBig(), delegatorRewards.ToBig())
	}

	if TotalDagBlocksCountCheck != rewardsStats.TotalDagBlocksCount {
		errorString := fmt.Sprintf("TotalDagBlocksCount (%d) based on validators stats != rewardsStats.TotalDagBlocksCount (%d)", TotalDagBlocksCountCheck, rewardsStats.TotalDagBlocksCount)
		fmt.Println(errorString)
	}

	if totalVoteWeightCheck != rewardsStats.TotalVotesWeight {
		errorString := fmt.Sprintf("TotalVotesWeight (%d) based on validators stats != rewardsStats.TotalVotesWeight (%d)", totalVoteWeightCheck, rewardsStats.TotalVotesWeight)
		fmt.Println(errorString)
	}

	if newMintedRewards.Cmp(blockReward) == 1 {
		errorString := fmt.Sprintf("newMintedRewards (%d) is more then blockReward (%d)", newMintedRewards, blockReward)
		fmt.Println(errorString)
	}

	self.storage.AddBalance(contract_address, totalReward.ToBig())

	return newMintedRewards
}

func (self *Contract) delegate_update_values(ctx vm.CallFrame, validator *Validator, prev_vote_count uint64) {
	validator.TotalStake.Add(validator.TotalStake, ctx.Value)
	v, _ := uint256.FromBig(ctx.Value)
	self.amount_delegated.Add(self.amount_delegated, v)
	new_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)

	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creates new state if necessary
func (self *Contract) delegate(ctx vm.CallFrame, block types.BlockNum, args sol.ValidatorAddressArgs) error {
	fmt.Println("delegate start")
	validator := self.validators.GetValidator(&args.Validator)
	validator_rewards := self.validators.GetValidatorRewards(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	if self.dpos_config.ValidatorMaximumStake.Cmp(bigutil.Add(ctx.Value, validator.TotalStake)) == -1 {
		return ErrValidatorsMaxStakeExceeded
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil && self.dpos_config.MinimumDeposit.Cmp(ctx.Value) == 1 {
		return ErrInsufficientDelegation
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = old_state.RewardsPer1Stake
		if validator.TotalStake.Cmp(big.NewInt(0)) > 0 {
			state.RewardsPer1Stake.Add(state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards.RewardsPool, validator.TotalStake))
		}
		validator_rewards.RewardsPool = big.NewInt(0)
		validator.LastUpdated = block
		state.Count++
	}

	prev_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)

	if delegation == nil {
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator, block, ctx.Value)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

		reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		if reward.Cmp(big.NewInt(0)) > 0 {
			transferContractBalance(&ctx, reward)
			self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.Validator, reward))
		}

		delegation.Stake.Add(delegation.Stake, ctx.Value)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)
	}

	self.delegate_update_values(ctx, validator, prev_vote_count)

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
	self.validators.ModifyValidatorRewards(&args.Validator, validator_rewards)
	self.evm.AddLog(self.logs.MakeDelegatedLog(ctx.CallerAccount.Address(), &args.Validator, ctx.Value))

	return nil
}

// Removes delegation from specified validator and claims rewards
// new undelegation object is created and moved to queue where after expiration can be claimed
func (self *Contract) undelegate(ctx vm.CallFrame, block types.BlockNum, args sol.UndelegateArgs) error {
	if self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrExistentUndelegation
	}

	validator := self.validators.GetValidator(&args.Validator)
	validator_rewards := self.validators.GetValidatorRewards(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		return ErrNonExistentDelegation
	}

	if delegation.Stake.Cmp(args.Amount) == -1 {
		return ErrInsufficientDelegation
	}

	if delegation.Stake.Cmp(args.Amount) != 0 && self.dpos_config.MinimumDeposit.Cmp(bigutil.Sub(delegation.Stake, args.Amount)) == 1 {
		return ErrInsufficientDelegation
	}

	prev_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards.RewardsPool, validator.TotalStake))
		validator_rewards.RewardsPool = big.NewInt(0)
		validator.LastUpdated = block
		state.Count++
	}

	// We need to claim rewards first
	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
	// Reward needs to be add to callers accounts as only stake is locked
	reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
	if reward.Cmp(big.NewInt(0)) > 0 {
		transferContractBalance(&ctx, reward)
		self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.Validator, reward))
	}

	// Creating undelegation request
	self.undelegations.CreateUndelegation(ctx.CallerAccount.Address(), &args.Validator, block+uint64(self.dpos_config.DelegationLockingPeriod), args.Amount)
	delegation.Stake.Sub(delegation.Stake, args.Amount)
	validator.TotalStake.Sub(validator.TotalStake, args.Amount)
	validator.UndelegationsCount++

	if delegation.Stake.Cmp(big.NewInt(0)) == 0 {
		self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.Validator)
	} else {
		delegation.LastUpdated = block
		state.Count++
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)
	}

	a, _ := uint256.FromBig(args.Amount)
	self.amount_delegated.Sub(self.amount_delegated, a)
	new_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	// We can delete validator object as it doesn't have any stake anymore (only before the hardfork)
	if !self.isMagnoliaHardfork(block) && validator.TotalStake.Cmp(big.NewInt(0)) == 0 && validator_rewards.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0 {
		self.validators.DeleteValidator(&args.Validator)
		self.state_put(&state_k, nil)
	} else {
		self.state_put(&state_k, state)
		self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
		self.validators.ModifyValidatorRewards(&args.Validator, validator_rewards)
	}
	self.evm.AddLog(self.logs.MakeUndelegatedLog(ctx.CallerAccount.Address(), &args.Validator, args.Amount))

	return nil
}

// Removes undelegation from queue and moves staked tokens back to delegator
// This only works after lock-up period expires
func (self *Contract) confirmUndelegate(ctx vm.CallFrame, block types.BlockNum, args sol.ValidatorAddressArgs) error {
	if !self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrNonExistentUndelegation
	}

	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	if undelegation.Block > block {
		return ErrLockedUndelegation
	}

	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	if self.isMagnoliaHardfork(block) {
		validator := self.validators.GetValidator(&args.Validator)
		if validator == nil {
			return ErrNonExistentValidator
		}
		validator.UndelegationsCount--

		validator_rewards := self.validators.GetValidatorRewards(&args.Validator)

		if validator.UndelegationsCount == 0 && validator.TotalStake.Cmp(big.NewInt(0)) == 0 && validator_rewards.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0 {
			self.validators.DeleteValidator(&args.Validator)
			self.state_delete(args.Validator[:], BlockToBytes(block))
		}
	}

	// TODO slashing of balance
	transferContractBalance(&ctx, undelegation.Amount)
	self.evm.AddLog(self.logs.MakeUndelegateConfirmedLog(ctx.CallerAccount.Address(), &args.Validator, undelegation.Amount))

	return nil
}

// Removes the undelegation request from queue and returns delegation value back to validator if possible
func (self *Contract) cancelUndelegate(ctx vm.CallFrame, block types.BlockNum, args sol.ValidatorAddressArgs) error {
	if !self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrNonExistentUndelegation
	}
	validator := self.validators.GetValidator(&args.Validator)
	validator_rewards := self.validators.GetValidatorRewards(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}
	prev_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)

	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards.RewardsPool, validator.TotalStake))
		validator_rewards.RewardsPool = big.NewInt(0)
		validator.LastUpdated = block
		state.Count++
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator, block, undelegation.Amount)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

		reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		if reward.Cmp(big.NewInt(0)) > 0 {
			transferContractBalance(&ctx, reward)
			self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.Validator, reward))
		}

		delegation.Stake.Add(delegation.Stake, undelegation.Amount)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)
	}
	validator.TotalStake.Add(validator.TotalStake, undelegation.Amount)
	validator.UndelegationsCount--

	a, _ := uint256.FromBig(undelegation.Amount)
	self.amount_delegated.Add(self.amount_delegated, a)
	new_vote_count := voteCount(validator.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
	self.validators.ModifyValidatorRewards(&args.Validator, validator_rewards)
	self.evm.AddLog(self.logs.MakeUndelegateCanceledLog(ctx.CallerAccount.Address(), &args.Validator, undelegation.Amount))

	return nil
}

// Moves delegated tokens from one delegator to another
func (self *Contract) redelegate(ctx vm.CallFrame, block types.BlockNum, args sol.RedelegateArgs) error {
	validator_from := self.validators.GetValidator(&args.ValidatorFrom)
	validator_rewards_from := self.validators.GetValidatorRewards(&args.ValidatorFrom)
	if validator_from == nil {
		return ErrNonExistentValidator
	}

	validator_to := self.validators.GetValidator(&args.ValidatorTo)
	validator_rewards_to := self.validators.GetValidatorRewards(&args.ValidatorTo)
	if validator_to == nil {
		return ErrNonExistentValidator
	}

	if self.dpos_config.ValidatorMaximumStake.Cmp(big.NewInt(0)) != 0 && self.dpos_config.ValidatorMaximumStake.Cmp(bigutil.Add(args.Amount, validator_to.TotalStake)) == -1 {
		return ErrValidatorsMaxStakeExceeded
	}

	prev_vote_count_from := voteCount(validator_from.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
	prev_vote_count_to := voteCount(validator_to.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
	//First we undelegate
	{
		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom)
		if delegation == nil {
			return ErrNonExistentDelegation
		}

		if delegation.Stake.Cmp(args.Amount) == -1 {
			return ErrInsufficientDelegation
		}

		if delegation.Stake.Cmp(args.Amount) != 0 && self.dpos_config.MinimumDeposit.Cmp(bigutil.Sub(delegation.Stake, args.Amount)) == 1 {
			return ErrInsufficientDelegation
		}

		state, state_k := self.state_get(args.ValidatorFrom[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(validator_from.LastUpdated))
			state = new(State)
			state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards_from.RewardsPool, validator_from.TotalStake))
			validator_rewards_from.RewardsPool = big.NewInt(0)
			validator_from.LastUpdated = block
			state.Count++
		}
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

		reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		if reward.Cmp(big.NewInt(0)) > 0 {
			transferContractBalance(&ctx, reward)
			self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.ValidatorFrom, reward))
		}

		delegation.Stake.Sub(delegation.Stake, args.Amount)
		validator_from.TotalStake.Sub(validator_from.TotalStake, args.Amount)

		if delegation.Stake.Cmp(big.NewInt(0)) == 0 {
			self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom)
		} else {
			delegation.LastUpdated = block
			state.Count++
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom, delegation)
		}

		if validator_from.TotalStake.Cmp(big.NewInt(0)) == 0 && validator_rewards_from.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0 {
			if !self.isMagnoliaHardfork(block) || validator_from.UndelegationsCount == 0 {
				self.validators.DeleteValidator(&args.ValidatorFrom)
				self.state_put(&state_k, nil)
			}
		} else {
			self.state_put(&state_k, state)
			self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.ValidatorFrom, validator_from)
			self.validators.ModifyValidatorRewards(&args.ValidatorFrom, validator_rewards_from)
		}

		new_vote_count := voteCount(validator_from.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
		if prev_vote_count_from != new_vote_count {
			self.eligible_vote_count -= prev_vote_count_from
			self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
		}

	}

	// Now we delegate
	state, state_k := self.state_get(args.ValidatorTo[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.ValidatorTo[:], BlockToBytes(validator_to.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards_to.RewardsPool, validator_to.TotalStake))
		validator_rewards_to.RewardsPool = big.NewInt(0)
		validator_to.LastUpdated = block
		state.Count++
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo)

	if delegation == nil {
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo, block, args.Amount)
		validator_to.TotalStake.Add(validator_to.TotalStake, args.Amount)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.ValidatorTo[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

		reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		if reward.Cmp(big.NewInt(0)) > 0 {
			transferContractBalance(&ctx, reward)
			self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.ValidatorTo, reward))
		}

		delegation.Stake.Add(delegation.Stake, args.Amount)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo, delegation)

		validator_to.TotalStake.Add(validator_to.TotalStake, args.Amount)
	}

	new_vote_count := voteCount(validator_to.TotalStake, self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
	if prev_vote_count_to != new_vote_count {
		self.eligible_vote_count -= prev_vote_count_to
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.ValidatorTo, validator_to)
	self.validators.ModifyValidatorRewards(&args.ValidatorTo, validator_rewards_to)
	self.evm.AddLog(self.logs.MakeRedelegatedLog(ctx.CallerAccount.Address(), &args.ValidatorFrom, &args.ValidatorTo, args.Amount))
	return nil
}

// Pays off accumulated rewards back to delegator address
func (self *Contract) claimRewards(ctx vm.CallFrame, block types.BlockNum, args sol.ValidatorAddressArgs) error {
	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		return ErrNonExistentDelegation
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		validator := self.validators.GetValidator(&args.Validator)
		validator_rewards := self.validators.GetValidatorRewards(&args.Validator)
		if validator == nil {
			return ErrNonExistentValidator
		}
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards.RewardsPool, validator.TotalStake))
		validator_rewards.RewardsPool = big.NewInt(0)
		validator.LastUpdated = block
		state.Count++
		self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
		self.validators.ModifyValidatorRewards(&args.Validator, validator_rewards)
	}

	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

	reward := self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
	if reward.Cmp(big.NewInt(0)) > 0 {
		transferContractBalance(&ctx, reward)
		self.evm.AddLog(self.logs.MakeRewardsClaimedLog(ctx.CallerAccount.Address(), &args.Validator, reward))
	}

	delegation.LastUpdated = block
	self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

	state.Count++
	self.state_put(&state_k, state)

	return nil
}

// Pays off accumulated rewards back to delegator address from multiple validators at a time
func (self *Contract) claimAllRewards(ctx vm.CallFrame, block types.BlockNum, args sol.ClaimAllRewardsArgs) (end bool, err error) {
	delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(ctx.CallerAccount.Address(), args.Batch, ClaimAllRewardsMaxCount)
	var tmp_claim_rewards_args sol.ValidatorAddressArgs
	for _, validator_address := range delegator_validators_addresses {
		tmp_claim_rewards_args.Validator = validator_address

		tmp_err := self.claimRewards(ctx, block, tmp_claim_rewards_args)
		if tmp_err != nil {
			err = util.ErrorString(tmp_err.Error() + " -> validator: " + validator_address.String())
			return
		}
	}

	err = nil
	return
}

// Pays off rewards from commission back to validator owner address
func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, block types.BlockNum, args sol.ValidatorAddressArgs) error {
	if !self.validators.CheckValidatorOwner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	validator := self.validators.GetValidator(&args.Validator)
	validator_rewards := self.validators.GetValidatorRewards(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	transferContractBalance(&ctx, validator_rewards.CommissionRewardsPool)
	self.evm.AddLog(self.logs.MakeCommissionRewardsClaimedLog(ctx.CallerAccount.Address(), &args.Validator, validator_rewards.CommissionRewardsPool))
	validator_rewards.CommissionRewardsPool = big.NewInt(0)

	if validator.TotalStake.Cmp(big.NewInt(0)) == 0 {
		self.validators.DeleteValidator(&args.Validator)
		self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
	} else {
		self.validators.ModifyValidatorRewards(&args.Validator, validator_rewards)
	}

	return nil
}

func validateProof(proof []byte, validator *common.Address) error {
	if len(proof) != 65 {
		return ErrWrongProof
	}

	// Make sure the public key is a valid one
	pubKey, err := crypto.Ecrecover(keccak256.Hash(validator.Bytes()).Bytes(), append(proof[:64], proof[64]-27))
	if err != nil {
		return err
	}

	// the first byte of pubkey is bitcoin heritage
	if common.BytesToAddress(keccak256.Hash(pubKey[1:])[12:]) != *validator {
		return ErrWrongProof
	}

	return nil
}

// Creates a new validator object and delegates to it specific value of tokens
func (self *Contract) registerValidatorWithoutChecks(ctx vm.CallFrame, block types.BlockNum, args sol.RegisterValidatorArgs) error {
	// Limit size of description & endpoint
	if len(args.Endpoint) > MaxEndpointLength {
		return ErrMaxEndpointLengthExceeded
	}
	if len(args.Description) > MaxDescriptionLength {
		return ErrMaxDescriptionLengthExceeded
	}
	if len(args.VrfKey) != VrfKeyLength {
		return ErrWrongVrfKey
	}
	if MaxCommission < uint64(args.Commission) {
		return ErrCommissionOverflow
	}

	if self.validators.ValidatorExists(&args.Validator) {
		return ErrExistentValidator
	}

	owner_address := ctx.CallerAccount.Address()
	delegation := self.delegations.GetDelegation(owner_address, &args.Validator)
	if delegation != nil {
		// This could happen only due some serious logic bug
		return ErrExistentDelegation
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state != nil {
		return ErrBrokenState
	}

	if self.dpos_config.ValidatorMaximumStake.Cmp(ctx.Value) == -1 {
		return ErrValidatorsMaxStakeExceeded
	}

	state = new(State)
	state.RewardsPer1Stake = big.NewInt(0)

	// Creates validator related objects in storage
	validator := self.validators.CreateValidator(self.isMagnoliaHardfork(block), owner_address, &args.Validator, args.VrfKey, block, args.Commission, args.Description, args.Endpoint)
	state.Count++
	self.evm.AddLog(self.logs.MakeValidatorRegisteredLog(&args.Validator))

	if ctx.Value.Cmp(big.NewInt(0)) == 1 {
		self.evm.AddLog(self.logs.MakeDelegatedLog(owner_address, &args.Validator, ctx.Value))
		self.delegations.CreateDelegation(owner_address, &args.Validator, block, ctx.Value)
		self.delegate_update_values(ctx, validator, 0)
		self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
		state.Count++
	}

	self.state_put(&state_k, state)
	return nil
}

// Main part of logic from `registerValidator` method. Doesn't have a few checks that is not needed for validator creation from genesis
func (self *Contract) registerValidator(ctx vm.CallFrame, block types.BlockNum, args sol.RegisterValidatorArgs) error {
	if err := validateProof(args.Proof, &args.Validator); err != nil {
		return err
	}

	if self.dpos_config.MinimumDeposit.Cmp(ctx.Value) == 1 {
		return ErrInsufficientDelegation
	}

	return self.registerValidatorWithoutChecks(ctx, block, args)
}

// Changes validator specific field as endpoint or description
func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args sol.SetValidatorInfoArgs) error {
	// Limit size of description & endpoint
	if len(args.Endpoint) > MaxEndpointLength {
		return ErrMaxEndpointLengthExceeded
	}
	if len(args.Description) > MaxDescriptionLength {
		return ErrMaxDescriptionLengthExceeded
	}

	if !self.validators.CheckValidatorOwner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	validator_info := self.validators.GetValidatorInfo(&args.Validator)
	if validator_info == nil {
		return ErrNonExistentValidator
	}

	validator_info.Description = args.Description
	validator_info.Endpoint = args.Endpoint

	self.validators.ModifyValidatorInfo(&args.Validator, validator_info)
	self.evm.AddLog(self.logs.MakeValidatorInfoSetLog(&args.Validator))

	return nil
}

// Changes validator commission to new rate
func (self *Contract) setCommission(ctx vm.CallFrame, block types.BlockNum, args sol.SetCommissionArgs) error {
	if !self.validators.CheckValidatorOwner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	if MaxCommission < uint64(args.Commission) {
		return ErrCommissionOverflow
	}

	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	if self.dpos_config.CommissionChangeFrequency != 0 && uint64(self.dpos_config.CommissionChangeFrequency) > (block-validator.LastCommissionChange) {
		return ErrForbiddenCommissionChange
	}

	if self.dpos_config.CommissionChangeDelta != 0 && self.dpos_config.CommissionChangeDelta < getDelta(validator.Commission, args.Commission) {
		return ErrForbiddenCommissionChange
	}

	validator.Commission = args.Commission
	validator.LastCommissionChange = block
	self.validators.ModifyValidator(self.isMagnoliaHardfork(block), &args.Validator, validator)
	self.evm.AddLog(self.logs.MakeCommissionSetLog(&args.Validator, args.Commission))

	return nil
}

// Returns single validator object
func (self *Contract) getValidatorInfo(block types.BlockNum, args sol.ValidatorAddressArgs) (sol.DposInterfaceValidatorBasicInfo, error) {
	var result sol.DposInterfaceValidatorBasicInfo
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return result, ErrNonExistentValidator
	}

	validator_info := self.validators.GetValidatorInfo(&args.Validator)
	if validator_info == nil {
		// This should never happen
		panic("getValidators - unable to fetch validator info data")
	}

	validator_rewards := self.validators.GetValidatorRewards(&args.Validator)

	result.Commission = validator.Commission
	result.CommissionReward = validator_rewards.CommissionRewardsPool
	result.LastCommissionChange = validator.LastCommissionChange
	result.Owner = self.validators.GetValidatorOwner(&args.Validator)
	result.TotalStake = validator.TotalStake
	result.Endpoint = validator_info.Endpoint
	result.Description = validator_info.Description
	return result, nil
}

func (self *Contract) to_validator_data(block types.BlockNum, validator_address, owner common.Address) (validator_data sol.DposInterfaceValidatorData) {
	validator := self.validators.GetValidator(&validator_address)
	if validator == nil {
		// This should never happen
		panic("to_validator_data - unable to fetch validator data")
	}

	validator_info := self.validators.GetValidatorInfo(&validator_address)
	if validator_info == nil {
		// This should never happen
		panic("to_validator_data - unable to fetch validator info data")
	}
	validator_rewards := self.validators.GetValidatorRewards(&validator_address)

	validator_data.Account = validator_address
	validator_data.Info.Commission = validator.Commission
	validator_data.Info.CommissionReward = validator_rewards.CommissionRewardsPool
	validator_data.Info.LastCommissionChange = validator.LastCommissionChange
	validator_data.Info.Owner = self.validators.GetValidatorOwner(&validator_address)
	validator_data.Info.TotalStake = validator.TotalStake
	validator_data.Info.Endpoint = validator_info.Endpoint
	validator_data.Info.Description = validator_info.Description
	return validator_data
}

// Returns batch of validators
func (self *Contract) getValidators(block types.BlockNum, args sol.GetValidatorsArgs) (validators []sol.DposInterfaceValidatorData, end bool) {
	validators_addresses, end := self.validators.GetValidatorsAddresses(args.Batch, GetValidatorsMaxCount)

	// Reserve slice capacity
	validators = make([]sol.DposInterfaceValidatorData, 0, len(validators_addresses))

	for _, validator_address := range validators_addresses {
		validators = append(validators, self.to_validator_data(block, validator_address, self.validators.GetValidatorOwner(&validator_address)))
	}
	return
}

// Returns batch of validators for specified owner
func (self *Contract) getValidatorsFor(block types.BlockNum, args sol.GetValidatorsForArgs) (validators []sol.DposInterfaceValidatorData, end bool) {
	validators_addresses, _ := self.validators.GetValidatorsAddresses(0, self.validators.GetValidatorsCount())

	// Reserve slice capacity
	validators = make([]sol.DposInterfaceValidatorData, 0, GetValidatorsMaxCount)
	skipped := uint32(0)
	to_skip := args.Batch * GetValidatorsMaxCount
	full := false

	for _, validator_address := range validators_addresses {
		owner := self.validators.GetValidatorOwner(&validator_address)
		if owner != args.Owner {
			continue
		}
		if skipped < to_skip {
			skipped++
			continue
		}

		if !full {
			validators = append(validators, self.to_validator_data(block, validator_address, owner))
			full = len(validators) == GetValidatorsMaxCount
		} else { // we found one more owner validator that belongs to the next batch.
			end = false
			return
		}
	}
	end = true
	return
}

// Returns batch of delegations for specified address
func (self *Contract) getDelegations(block types.BlockNum, args sol.GetDelegationsArgs) (delegations []sol.DposInterfaceDelegationData, end bool) {
	delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(&args.Delegator, args.Batch, GetDelegationsMaxCount)

	// Reserve slice capacity
	delegations = make([]sol.DposInterfaceDelegationData, 0, len(delegator_validators_addresses))

	for _, validator_address := range delegator_validators_addresses {
		delegation := self.delegations.GetDelegation(&args.Delegator, &validator_address)
		validator := self.validators.GetValidator(&validator_address)
		if delegation == nil || validator == nil {
			// This should never happen
			panic("getDelegations - unable to fetch delegation data")
		}
		validator_rewards := self.validators.GetValidatorRewards(&validator_address)

		var delegation_data sol.DposInterfaceDelegationData
		delegation_data.Account = validator_address
		delegation_data.Delegation.Stake = delegation.Stake

		/// Temp values
		state, _ := self.state_get(validator_address[:], BlockToBytes(validator.LastUpdated))
		old_state, _ := self.state_get(validator_address[:], BlockToBytes(delegation.LastUpdated))
		if state == nil || old_state == nil {
			// This should never happen
			panic("getDelegations - unable to state data")
		}
		current_reward_per_stake := bigutil.Add(state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_rewards.RewardsPool, validator.TotalStake))
		reward_per_stake := bigutil.Sub(current_reward_per_stake, old_state.RewardsPer1Stake)
		////

		delegation_data.Delegation.Rewards = self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		delegations = append(delegations, delegation_data)
	}
	return
}

// Returns batch of undelegation from queue for given address
func (self *Contract) getUndelegations(args sol.GetUndelegationsArgs) (undelegations []sol.DposInterfaceUndelegationData, end bool) {
	undelegations_addresses, end := self.undelegations.GetDelegatorValidatorsAddresses(&args.Delegator, args.Batch, GetUndelegationsMaxCount)

	// Reserve slice capacity
	undelegations = make([]sol.DposInterfaceUndelegationData, 0, len(undelegations_addresses))

	for _, validator_address := range undelegations_addresses {
		undelegation := self.undelegations.GetUndelegation(&args.Delegator, &validator_address)
		if undelegation == nil {
			// This should never happen
			panic("getUndelegations - unable to fetch undelegation data")
		}

		var undelegation_data sol.DposInterfaceUndelegationData
		undelegation_data.Validator = validator_address
		// Validator can be already deleted before confirming undelegation if he had 0 rewards and stake balances
		undelegation_data.ValidatorExists = self.validators.ValidatorExists(&validator_address)
		undelegation_data.Stake = undelegation.Amount
		undelegation_data.Block = undelegation.Block

		undelegations = append(undelegations, undelegation_data)
	}

	return
}

func (self *Contract) state_get(validator_addr, block []byte) (state *State, key common.Hash) {
	key = stor_k_2(field_state, validator_addr, block)
	self.storage.Get(&key, func(bytes []byte) {
		state = new(State)
		rlp.MustDecodeBytes(bytes, state)
	})
	return
}

// Gets state object from storage and decrements it's count
// if number of references is 0 it also removes object from storage
func (self *Contract) state_get_and_decrement(validator_addr, block []byte) (state *State) {
	key := stor_k_1(field_state, validator_addr, block)
	self.storage.Get(key, func(bytes []byte) {
		state = new(State)
		rlp.MustDecodeBytes(bytes, state)
	})
	if state == nil {
		// This should never happen
		panic("state_get_and_decrement - unable to fetch undelegation data")
	}
	state.Count--
	if state.Count == 0 {
		self.state_put(key, nil)
	} else {
		self.state_put(key, state)
	}
	return
}

// Saves state object to storage
func (self *Contract) state_put(key *common.Hash, state *State) {
	if state != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(state))
	} else {
		self.storage.Put(key, nil)
	}
}

func (self *Contract) state_delete(validator_addr, block []byte) {
	key := stor_k_1(field_state, validator_addr, block)
	self.storage.Put(key, nil)
}

func (self *Contract) apply_genesis_entry(validator_info *chain_config.GenesisValidator, make_context func(caller *common.Address, value *big.Int) vm.CallFrame) {
	args := validator_info.Gen_register_validator_args()

	registrationError := self.registerValidatorWithoutChecks(make_context(&validator_info.Owner, big.NewInt(0)), 0, args)
	if registrationError != nil {
		panic("apply_genesis_entry: registrationError: " + registrationError.Error())
	}

	for delegator, amount := range validator_info.Delegations {
		// for delegate call with a transaction value(out delegation amount) is transferred with transaction logic
		// before entering this function. So we should do the same thing manually
		self.storage.SubBalance(&delegator, amount)
		self.storage.AddBalance(contract_address, amount)

		delegationError := self.delegate(make_context(&delegator, amount), 0, sol.ValidatorAddressArgs{validator_info.Address})
		if delegationError != nil {
			panic("apply_genesis_entry: delegationError: " + delegationError.Error())
		}
	}
}

func (self *Contract) calculateRewardPer1Stake(rewardsPool *big.Int, stake *big.Int) *big.Int {
	return bigutil.Div(bigutil.Mul(rewardsPool, self.dpos_config.ValidatorMaximumStake), stake)
}

func (self *Contract) calculateDelegatorReward(rewardPer1Stake *big.Int, stake *big.Int) *big.Int {
	return bigutil.Div(bigutil.Mul(rewardPer1Stake, stake), self.dpos_config.ValidatorMaximumStake)
}

func (self *Contract) isMagnoliaHardfork(block types.BlockNum) bool {
	return block >= self.hardforks_config.MagnoliaHfBlockNum
}

func transferContractBalance(ctx *vm.CallFrame, balance *big.Int) {
	// ctx.Account == contract address
	// ctx.CallerAccount == caller address
	if availableBalance := ctx.Account.GetBalance(); availableBalance.Cmp(balance) == -1 {
		errorString := fmt.Sprintf("Contract balance (%d) is smaller than required amount (%d)", availableBalance, balance)
		panic(errorString)
	}
	ctx.Account.SubBalance(balance)
	ctx.CallerAccount.AddBalance(balance)
}

// Returns block number as bytes
func BlockToBytes(number types.BlockNum) []byte {
	big := new(big.Int)
	big.SetUint64(number)
	return big.Bytes()
}

// Calculates vote count from staking balance based on config values
func voteCount(staking_balance, eligibility_threshold, vote_eligibility_balance_step *big.Int) uint64 {
	tmp := big.NewInt(0)
	if staking_balance.Cmp(eligibility_threshold) >= 0 {
		tmp.Div(staking_balance, vote_eligibility_balance_step)
	}
	asserts.Holds(tmp.IsUint64())
	return tmp.Uint64()
}

// Safe add64, that panics on overflow (should never happen - misconfiguration)
func add64p(a, b uint64) uint64 {
	c := a + b
	if c < a || c < b {
		panic("addition overflow " + strconv.FormatUint(a, 10) + " " + strconv.FormatUint(b, 10))
	}
	return c
}

// Returns absolute value of the difference of two uint16
func getDelta(x, y uint16) uint16 {
	if x < y {
		return y - x
	} else {
		return x - y
	}
}

func Max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}
