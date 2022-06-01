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

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"

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

// Error values
var ErrInsufficientBalance = util.ErrorString("Insufficient balance")
var ErrNonExistentValidator = util.ErrorString("Validator does not exist")
var ErrNonExistentDelegation = util.ErrorString("Delegation does not exist")
var ErrExistentUndelegation = util.ErrorString("Undelegation already exist")
var ErrNonExistentUndelegation = util.ErrorString("Undelegation does not exist")
var ErrLockedUndelegation = util.ErrorString("Undelegation is not yet ready to be withdrawn")
var ErrExistentValidator = util.ErrorString("Validator already exist")
var ErrBrokenState = util.ErrorString("Fatal error state is broken")
var ErrValidatorsMaxStakeExceeded = util.ErrorString("Validator's max stake exceeded")
var ErrInsufficientDelegation = util.ErrorString("Insufficient delegation")
var ErrCallIsNotToplevel = util.ErrorString("only top-level calls are allowed")
var ErrWrongProof = util.ErrorString("Wrong proof, validator address could not be recoverd")
var ErrWrongOwnerAcc = util.ErrorString("This account is not owner of specified validator")
var ErrForbiddenCommissionChange = util.ErrorString("Forbidden commission change")
var ErrMaxEndpointLengthExceeded = util.ErrorString("Max endpoint length exceeded")
var ErrMaxDescriptionLengthExceeded = util.ErrorString("Max description length exceeded")

// Contract storage fields keys
var (
	field_validators    = []byte{0}
	field_state         = []byte{1}
	field_delegations   = []byte{2}
	field_undelegations = []byte{3}

	field_eligible_vote_count = []byte{4}
	field_amount_delegated    = []byte{5}
)

// Rewards related constants
// TODO: these params will be propagated through config
var (
	TaraPrecision   = big.NewInt(1e+18)              // Tara precision
	YieldPercentage = big.NewInt(20)                 // 20% yield
	BlocksPerYear   = big.NewInt(365 * 24 * 60 * 15) // 365 days * 24 hours * 60 minutes * 15 (1 pbft block every 4 seconds -> 15 per minute)
)

// const value of 10000 so we do not need to allocate it again
var Big10000 = big.NewInt(10000)
var Big100 = big.NewInt(100)

// Max num of characters in url
const MaxEndpointLength = 50

// Max num of characters in description
const MaxDescriptionLength = 100

// Maximum number of validators per batch returned by getValidators call
const GetValidatorsMaxCount = 50

// Maximum number of validators per batch returned by getDelegatorDelegations call
const GetDelegatorDelegationsMaxCount = 50

// State of the rewards distribution algorithm
type State struct {
	// represents number of rewards per 1 stake
	RewardsPer1Stake *big.Int
	// number of references
	Count uint32
}

// Main contract class
type Contract struct {
	cfg Config
	// current storage
	storage StorageWrapper
	// delayed storage for PBFT
	delayedStorage Reader
	// ABI of the contract
	Abi abi.ABI

	// Iterable storages
	validators    Validators
	delegations   Delegations
	undelegations Undelegations

	// values for PBFT
	eligible_vote_count_orig uint64
	eligible_vote_count      uint64
	amount_delegated_orig    *big.Int
	amount_delegated         *big.Int

	lazy_init_done bool
}

// Initialize contract class
func (self *Contract) Init(cfg Config, storage Storage, readStorage Reader) *Contract {
	self.cfg = cfg
	self.storage.Init(storage)
	self.delayedStorage = readStorage
	return self
}

// Updates delayted storage after each commited block
func (self *Contract) UpdateStorage(readStorage Reader) {
	self.delayedStorage = readStorage
}

// Updates config - for HF
func (self *Contract) UpdateConfig(cfg Config) {
	self.cfg = cfg
}

// Register this precompiled contract
func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

// Calculate required gas for call to this contract
func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	// TODO: based on method being called, calculate the gas somehow. Doing it based on the length of input is totally useless
	return uint64(len(ctx.Input)) * 20
}

// Lazy initialization only if contract is needed
func (self *Contract) lazy_init() {
	if self.lazy_init_done {
		return
	}
	self.lazy_init_done = true

	self.Abi, _ = abi.JSON(strings.NewReader(TaraxaDposClientMetaData))

	self.validators.Init(&self.storage, field_validators)
	self.delegations.Init(&self.storage, field_delegations)
	self.undelegations.Init(&self.storage, field_undelegations)

	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		self.eligible_vote_count_orig = bin.DEC_b_endian_compact_64(bytes)
	})
	self.eligible_vote_count = self.eligible_vote_count_orig

	self.amount_delegated_orig = bigutil.Big0
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		self.amount_delegated_orig = bigutil.FromBytes(bytes)
	})
	self.amount_delegated = self.amount_delegated_orig
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
		self.amount_delegated_orig = self.amount_delegated
	}
}

// Should be called on each block commit - updates delayedStorage
func (self *Contract) CommitCall(readStorage Reader) {
	defer self.storage.ClearCache()
	// Storage Update
	self.delayedStorage = readStorage
}

// Fills contract based on genesis values
func (self *Contract) ApplyGenesis() error {
	self.lazy_init()

	for _, entry := range self.cfg.GenesisState {
		self.apply_genesis_entry(&entry.Benefactor, entry.Transfers)
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
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse delegate input args: ", err)
			return nil, err
		}

		return nil, self.delegate(ctx, evm.GetBlock().Number, args)

	case "undelegate":
		var args UndelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse delegate input args: ", err)
			return nil, err
		}

		return nil, self.undelegate(ctx, evm.GetBlock().Number, args)

	case "confirmUndelegate":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse confirmUndelegate input args: ", err)
			return nil, err
		}

		return nil, self.confirmUndelegate(ctx, evm.GetBlock().Number, args)

	case "cancelUndelegate":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse cancelUndelegate input args: ", err)
			return nil, err
		}

		return nil, self.cancelUndelegate(ctx, evm.GetBlock().Number, args)

	case "reDelegate":
		var args RedelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse reDelegate input args: ", err)
			return nil, err
		}

		return nil, self.redelegate(ctx, evm.GetBlock().Number, args)

	case "claimRewards":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimRewards input args: ", err)
			return nil, err
		}

		return nil, self.claimRewards(ctx, evm.GetBlock().Number, args)

	case "claimCommissionRewards":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimCommissionRewards input args: ", err)
			return nil, err
		}
		return nil, self.claimCommissionRewards(ctx, evm.GetBlock().Number, args)

	case "setCommission":
		var args SetCommissionArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimCommissionRewards input args: ", err)
			return nil, err
		}
		return nil, self.setCommission(ctx, evm.GetBlock().Number, args)

	case "registerValidator":
		var args RegisterValidatorArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse registerValidator input args: ", err)
			return nil, err
		}

		return nil, self.registerValidator(ctx, evm.GetBlock().Number, args)

	case "setValidatorInfo":
		var args SetValidatorInfoArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse setValidatorInfo input args: ", err)
			return nil, err
		}

		return nil, self.setValidatorInfo(ctx, args)

	case "isValidatorEligible":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse isValidatorEligible input args: ", err)
			return nil, err
		}
		result := self.delayedStorage.IsEligible(&args.Validator)
		return method.Outputs.Pack(result)

	case "getTotalEligibleVotesCount":
		result := self.delayedStorage.EligibleVoteCount()
		return method.Outputs.Pack(result)

	case "getValidatorEligibleVotesCount":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}

		result := self.delayedStorage.GetEligibleVoteCount(&args.Validator)
		return method.Outputs.Pack(result)

	case "getValidators":
		var args GetValidatorsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidators input args: ", err)
			return nil, err
		}

		result := self.getValidators(args)
		return method.Outputs.Pack(result)

	case "getDelegatorDelegations":
		var args GetDelegatorDelegationsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getDelegatorDelegations input args: ", err)
			return nil, err
		}

		result := self.getDelegatorDelegations(args)
		return method.Outputs.Pack(result)

	case "getUndelegations":
		var args GetDelegatorDelegationsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getUndelegations input args: ", err)
			return nil, err
		}

		result := self.getUndelegations(args)
		return method.Outputs.Pack(result)
	}

	return nil, nil
}

func (self *Contract) DistributeRewards(rewardsStats *rewards_stats.RewardsStats, feesRewards *FeesRewards) {
	// When calling DistributeRewards, internal structures must be always initialized
	self.lazy_init()

	// Calculates number of tokens to be generated as block reward
	blockReward := bigutil.Mul(self.amount_delegated, YieldPercentage)
	blockReward = bigutil.Div(blockReward, bigutil.Mul(Big100, BlocksPerYear))

	totalUniqueTxsCountCheck := uint32(0)

	// Calculates validators rewards
	for validatorAddress, validatorStats := range rewardsStats.ValidatorsStats {
		totalUniqueTxsCountCheck += validatorStats.UniqueTxsCount

		validator := self.validators.GetValidator(&validatorAddress)
		if validator == nil {
			// This should never happen. Validator must exist(be eligible) either now or at least in delayed storage
			if !self.delayedStorage.IsEligible(&validatorAddress) {
				panic("update_rewards - non existent validator")
			}

			// This could happen due to few blocks artificial delay we use to determine if validator is eligible or not when
			// checking it during consesnus. If everyone undelegates from validator and also he claims his commission rewards
			// during the the period of time, which is < then delay we use, he is deleted from contract storage, but he will be
			// able to propose few more blocks. This situation is extremly unlikely, but technically possible.
			// If it happens, valdiator will simply not receive rewards for those few last blocks/votes he produced
			continue
		}

		// Calculate it like this to eliminate rounding error as much as possible
		validatorReward := bigutil.Mul(big.NewInt(int64(validatorStats.UniqueTxsCount)), blockReward)
		validatorReward = bigutil.Div(validatorReward, big.NewInt(int64(rewardsStats.TotalUniqueTxsCount)))

		// TODO: once we have also votes statistics, use it in calculations
		// TODO: should voter with 1M stake be rewarded same as voter with 10M stake ???

		// Adds fees for all txs that validator added in his blocks as first
		validatorReward = bigutil.Add(validatorReward, feesRewards.GetTxsFeesReward(validatorAddress))

		validatorCommission := bigutil.Div(bigutil.Mul(validatorReward, big.NewInt(int64(validator.Commission))), Big10000)
		delegatorsRewards := bigutil.Sub(validatorReward, validatorCommission)

		validator.CommissionRewardsPool = bigutil.Add(validator.CommissionRewardsPool, validatorCommission)
		validator.RewardsPool = bigutil.Add(validator.RewardsPool, delegatorsRewards)

		self.validators.ModifyValidator(&validatorAddress, validator)
	}

	// TODO: debug check - can be deleted for release
	if totalUniqueTxsCountCheck != rewardsStats.TotalUniqueTxsCount {
		errorString := fmt.Sprintf("TotalUniqueTxsCount (%d) based on validators stats != rewardsStats.TotalUniqueTxsCount (%d)", totalUniqueTxsCountCheck, rewardsStats.TotalUniqueTxsCount)
		panic(errorString)
	}
}

// Delegates specified number of tokens to specified validator and creates new delegation object
// It also increase total stake of specified validator and creas new state if necessary
func (self *Contract) delegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	if self.cfg.MaximumStake.Cmp(bigutil.Big0) != 0 && self.cfg.MaximumStake.Cmp(bigutil.Add(ctx.Value, validator.TotalStake)) == -1 {
		return ErrValidatorsMaxStakeExceeded
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil && self.cfg.MinimumDeposit.Cmp(ctx.Value) == 1 {
		return ErrInsufficientDelegation
	}

	prev_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	if delegation == nil {
		// ctx.Account == contract address. Substract tokens that were sent to the contract as delegation
		ctx.Account.SubBalance(ctx.Value)
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator, block, ctx.Value)
		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)

		// ctx.CallerAccount == caller address
		ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))
		// ctx.Account == contract address
		ctx.Account.SubBalance(ctx.Value)

		delegation.Stake = bigutil.Add(delegation.Stake, ctx.Value)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value)
	}

	self.amount_delegated = bigutil.Add(self.amount_delegated, ctx.Value)
	new_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(&args.Validator, validator)
	return nil
}

// Removes delegation from specified validator and claims rewards
// new undelegation object is created and moved to queue where after expiration can be claimed
func (self *Contract) undelegate(ctx vm.CallFrame, block types.BlockNum, args UndelegateArgs) error {
	if self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrExistentUndelegation
	}

	validator := self.validators.GetValidator(&args.Validator)
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

	if delegation.Stake.Cmp(args.Amount) != 0 && self.cfg.MinimumDeposit.Cmp(bigutil.Sub(delegation.Stake, args.Amount)) == 1 {
		return ErrInsufficientDelegation
	}

	prev_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	// We need to claim rewards first
	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
	// Reward needs to be add to callers accounts as only stake is locked
	ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))

	// Creating undelegation request
	self.undelegations.CreateUndelegation(ctx.CallerAccount.Address(), &args.Validator, block+self.cfg.WithdrawalDelay, args.Amount)
	delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
	validator.TotalStake = bigutil.Sub(validator.TotalStake, args.Amount)

	if delegation.Stake.Cmp(bigutil.Big0) == 0 {
		self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.Validator)
	} else {
		delegation.LastUpdated = block
		state.Count++
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)
	}

	self.amount_delegated = bigutil.Sub(self.amount_delegated, args.Amount)
	new_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	// We can delete validator object as it doesn't have any stake anymore'
	if validator.TotalStake.Cmp(bigutil.Big0) == 0 && validator.CommissionRewardsPool.Cmp(bigutil.Big0) == 0 {
		self.validators.DeleteValidator(&args.Validator)
		self.state_put(&state_k, nil)
	} else {
		self.state_put(&state_k, state)
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	return nil
}

// Removes undelegation from queue and moves staked toknes back to delegator
// This only works after lock-up period expires
func (self *Contract) confirmUndelegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	if !self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrNonExistentUndelegation
	}
	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	if undelegation.Block > block {
		return ErrLockedUndelegation
	}
	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	// TODO slashing of balance
	ctx.CallerAccount.AddBalance(undelegation.Amount)
	return nil
}

// Removes the undelegation request from queue and returns delegation value back to validator if possible
func (self *Contract) cancelUndelegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	if !self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrNonExistentUndelegation
	}
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}
	prev_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator, block, undelegation.Amount)
		validator.TotalStake = bigutil.Add(validator.TotalStake, undelegation.Amount)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
		ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))

		delegation.Stake = bigutil.Add(delegation.Stake, undelegation.Amount)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, undelegation.Amount)
	}
	self.amount_delegated = bigutil.Add(self.amount_delegated, undelegation.Amount)
	new_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(&args.Validator, validator)
	return nil
}

// Moves delegated tokens from one delegator to another
func (self *Contract) redelegate(ctx vm.CallFrame, block types.BlockNum, args RedelegateArgs) error {
	validator_from := self.validators.GetValidator(&args.ValidatorFrom)
	if validator_from == nil {
		return ErrNonExistentValidator
	}

	validator_to := self.validators.GetValidator(&args.ValidatorTo)
	if validator_to == nil {
		return ErrNonExistentValidator
	}

	if self.cfg.MaximumStake.Cmp(bigutil.Big0) != 0 && self.cfg.MaximumStake.Cmp(bigutil.Add(args.Amount, validator_to.TotalStake)) == -1 {
		return ErrValidatorsMaxStakeExceeded
	}

	prev_vote_count_from := voteCount(validator_from.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	prev_vote_count_to := voteCount(validator_to.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	//First we undelegate
	{
		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom)
		if delegation == nil {
			return ErrNonExistentDelegation
		}

		if delegation.Stake.Cmp(args.Amount) == -1 {
			return ErrInsufficientDelegation
		}

		if delegation.Stake.Cmp(args.Amount) != 0 && self.cfg.MinimumDeposit.Cmp(bigutil.Sub(delegation.Stake, args.Amount)) == 1 {
			return ErrInsufficientDelegation
		}

		state, state_k := self.state_get(args.ValidatorFrom[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(validator_from.LastUpdated))
			state = new(State)
			state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_from.RewardsPool, validator_from.TotalStake))
			validator_from.RewardsPool = bigutil.Big0
			validator_from.LastUpdated = block
			state.Count++
		}
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
		ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))

		delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
		validator_from.TotalStake = bigutil.Sub(validator_from.TotalStake, args.Amount)

		if delegation.Stake.Cmp(bigutil.Big0) == 0 {
			self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom)
		} else {
			delegation.LastUpdated = block
			state.Count++
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom, delegation)
		}

		if validator_from.TotalStake.Cmp(bigutil.Big0) == 0 && validator_from.CommissionRewardsPool.Cmp(bigutil.Big0) == 0 {
			self.validators.DeleteValidator(&args.ValidatorFrom)
			self.state_put(&state_k, nil)
		} else {
			self.state_put(&state_k, state)
			self.validators.ModifyValidator(&args.ValidatorFrom, validator_from)
		}

		new_vote_count := voteCount(validator_from.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if prev_vote_count_from != new_vote_count {
			self.eligible_vote_count -= prev_vote_count_from
			self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
		}

	}

	// Now we delegate
	{
		state, state_k := self.state_get(args.ValidatorTo[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.ValidatorTo[:], BlockToBytes(validator_to.LastUpdated))
			state = new(State)
			state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator_to.RewardsPool, validator_to.TotalStake))
			validator_to.RewardsPool = bigutil.Big0
			validator_to.LastUpdated = block
			state.Count++
		}

		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo)

		if delegation == nil {
			self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo, block, args.Amount)
			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		} else {
			// We need to claim rewards first
			old_state := self.state_get_and_decrement(args.ValidatorTo[:], BlockToBytes(delegation.LastUpdated))
			reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
			ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))

			delegation.Stake = bigutil.Add(delegation.Stake, args.Amount)
			delegation.LastUpdated = block
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo, delegation)

			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		}

		new_vote_count := voteCount(validator_to.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if prev_vote_count_to != new_vote_count {
			self.eligible_vote_count -= prev_vote_count_to
			self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
		}

		state.Count++
		self.state_put(&state_k, state)
		self.validators.ModifyValidator(&args.ValidatorTo, validator_to)
	}
	return nil
}

// Pays off accumulated rewards back to delegator address
func (self *Contract) claimRewards(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		return ErrNonExistentDelegation
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		validator := self.validators.GetValidator(&args.Validator)
		if validator == nil {
			return ErrNonExistentValidator
		}
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
	ctx.CallerAccount.AddBalance(self.calculateDelegatorReward(reward_per_stake, delegation.Stake))

	delegation.LastUpdated = block
	self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

	state.Count++
	self.state_put(&state_k, state)

	return nil
}

// Pays off rewards from commission back to validator owner address
func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	if !self.validators.CheckValidatorOwner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	ctx.CallerAccount.AddBalance(validator.CommissionRewardsPool)
	validator.CommissionRewardsPool = bigutil.Big0

	if validator.TotalStake.Cmp(bigutil.Big0) == 0 {
		self.validators.DeleteValidator(&args.Validator)
		self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
	} else {
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	return nil
}

// Creates a new validator object and delegates to it specific value of tokens
func (self *Contract) registerValidator(ctx vm.CallFrame, block types.BlockNum, args RegisterValidatorArgs) error {
	// Limit size of description & endpoint
	if len(args.Endpoint) > MaxEndpointLength {
		return ErrMaxEndpointLengthExceeded
	}
	if len(args.Description) > MaxDescriptionLength {
		return ErrMaxDescriptionLengthExceeded
	}

	// Make sure the public key is a valid one
	pubKey, err := crypto.Ecrecover(args.Validator.Hash().Bytes(), args.Proof)
	// the first byte of pubkey is bitcoin heritage
	if err != nil {
		return err
	}

	if common.BytesToAddress(keccak256.Hash(pubKey[1:])[12:]) != args.Validator {
		return ErrWrongProof
	}

	if self.validators.ValidatorExists(&args.Validator) {
		return ErrExistentValidator
	}

	if self.cfg.MinimumDeposit.Cmp(ctx.Value) == 1 {
		return ErrInsufficientDelegation
	}

	owner_address := ctx.CallerAccount.Address()
	delegation := self.delegations.GetDelegation(owner_address, &args.Validator)
	if delegation != nil {
		// This could happen only due some serious logic bug
		panic("registerValidator: delegation already exists")
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state != nil {
		return ErrBrokenState
	}

	// ctx.Account == contract address. Substract tokens that were sent to the contract as delegation
	ctx.Account.SubBalance(ctx.Value)

	state = new(State)
	state.RewardsPer1Stake = bigutil.Big0

	// Creates validator related objects in storage
	self.validators.CreateValidator(owner_address, &args.Validator, block, ctx.Value, args.Commission, args.Description, args.Endpoint)
	state.Count++

	if ctx.Value.Cmp(bigutil.Big0) == 1 {
		new_vote_count := voteCount(ctx.Value, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if new_vote_count > 0 {
			self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
		}
		self.amount_delegated = bigutil.Add(self.amount_delegated, ctx.Value)
		// Creates Delegation object in storage
		self.delegations.CreateDelegation(owner_address, &args.Validator, block, ctx.Value)
		state.Count++
	}
	self.state_put(&state_k, state)

	return nil
}

// Changes validator specific field as endpoint or description
func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
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

	return nil
}

// Changes validator commission to new rate
func (self *Contract) setCommission(ctx vm.CallFrame, block types.BlockNum, args SetCommissionArgs) error {
	if !self.validators.CheckValidatorOwner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	if self.cfg.CommissionChangeFrequency != 0 && self.cfg.CommissionChangeFrequency > (block-validator.LastCommissionChange) {
		return ErrForbiddenCommissionChange
	}

	if self.cfg.CommissionChangeDelta != 0 && self.cfg.CommissionChangeDelta < getDelta(validator.Commission, args.Commission) {
		return ErrForbiddenCommissionChange
	}

	validator.Commission = args.Commission
	validator.LastCommissionChange = block
	self.validators.ModifyValidator(&args.Validator, validator)

	return nil
}

// Returns batch of validators
func (self *Contract) getValidators(args GetValidatorsArgs) (result GetValidatorsRet) {
	// TODO: measure performance of this call - if it is too bad -> decrease GetValidatorsMaxCount constant

	validators_addresses, end := self.validators.GetValidatorsAddresses(args.Batch, GetValidatorsMaxCount)

	// Reserve slice capacity
	result.Validators = make([]DposInterfaceValidatorData, 0, len(validators_addresses))

	for _, validator_address := range validators_addresses {
		validator := self.validators.GetValidator(&validator_address)
		if validator == nil {
			// This should never happen
			panic("getValidators - unable to fetch validator data")
		}

		validator_info := self.validators.GetValidatorInfo(&validator_address)
		if validator_info == nil {
			// This should never happen
			panic("getValidators - unable to fetch validator info data")
		}

		var validator_data DposInterfaceValidatorData
		validator_data.Account = validator_address
		validator_data.Info.Commission = validator.Commission
		validator_data.Info.CommissionReward = validator.CommissionRewardsPool
		validator_data.Info.TotalStake = validator.TotalStake
		validator_data.Info.Endpoint = validator_info.Endpoint
		validator_data.Info.Description = validator_info.Description

		result.Validators = append(result.Validators, validator_data)
	}

	result.End = end
	return
}

// Returns batch of delegations for specified address
func (self *Contract) getDelegatorDelegations(args GetDelegatorDelegationsArgs) (result GetDelegatorDelegationRet) {

	// TODO: measure performance of this call - if it is too bad -> decrease GetValidatorsMaxCount constant
	// TODO: this will be super expensice call probably

	delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(&args.Delegator, args.Batch, GetDelegatorDelegationsMaxCount)

	// Reserve slice capacity
	result.Delegations = make([]DposInterfaceDelegationData, 0, len(delegator_validators_addresses))

	for _, validator_address := range delegator_validators_addresses {
		delegation := self.delegations.GetDelegation(&args.Delegator, &validator_address)
		validator := self.validators.GetValidator(&validator_address)
		if delegation == nil || validator == nil {
			// This should never happen
			panic("getDelegatorDelegations - unable to fetch delegation data")
		}

		var delegation_data DposInterfaceDelegationData
		delegation_data.Account = validator_address
		delegation_data.Delegation.Stake = delegation.Stake

		/// Temp values
		state, _ := self.state_get(validator_address[:], BlockToBytes(validator.LastUpdated))
		old_state, _ := self.state_get(validator_address[:], BlockToBytes(validator.LastUpdated))
		if state == nil || old_state == nil {
			// This should never happen
			panic("getDelegatorDelegations - unable to state data")
		}
		current_reward_per_stake := bigutil.Add(state.RewardsPer1Stake, self.calculateRewardPer1Stake(validator.RewardsPool, validator.TotalStake))
		reward_per_stake := bigutil.Sub(current_reward_per_stake, old_state.RewardsPer1Stake)
		////

		delegation_data.Delegation.Rewards = self.calculateDelegatorReward(reward_per_stake, delegation.Stake)
		result.Delegations = append(result.Delegations, delegation_data)
	}

	result.End = end
	return
}

// Returns batch of undelegation from queue for given address
func (self *Contract) getUndelegations(args GetDelegatorDelegationsArgs) (result GetUnelegationsRet) {
	undelegations_addresses, end := self.undelegations.GetUndelegations(&args.Delegator, args.Batch, GetDelegatorDelegationsMaxCount)

	// Reserve slice capacity
	result.Undelegations = make([]DposInterfaceUndelegationData, 0, len(undelegations_addresses))

	for _, validator_address := range undelegations_addresses {
		undelegation := self.undelegations.GetUndelegation(&args.Delegator, &validator_address)
		if undelegation == nil {
			// This should never happen
			panic("getUndelegations - unable to fetch undelegation data")
		}

		var undelegation_data DposInterfaceUndelegationData
		undelegation_data.Validator = validator_address
		undelegation_data.Stake = undelegation.Amount
		undelegation_data.Block = undelegation.Block

		result.Undelegations = append(result.Undelegations, undelegation_data)
	}

	result.End = end
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

// Creates validator and delegation based on the given values
func (self *Contract) apply_genesis_entry(delegator_address *common.Address, transfers []GenesisTransfer) {
	// TODO fill them?
	var args RegisterValidatorArgs

	for _, delegation := range transfers {
		if self.cfg.MinimumDeposit.Cmp(delegation.Value) == 1 {
			panic("registerValidator: delegation is lower then the minimum")
		}
		if self.cfg.MaximumStake.Cmp(bigutil.Big0) != 0 && self.cfg.MaximumStake.Cmp(delegation.Value) == -1 {
			panic("registerValidator: delegation is lower then the minimum")
		}
		if delegation.Value.Cmp(bigutil.Big0) == 1 {
			var state *State
			var state_k common.Hash
			delegation_object := self.delegations.GetDelegation(delegator_address, &delegation.Beneficiary)
			if delegation_object != nil {
				panic("registerValidator: delegation already exists")
			}

			if self.validators.ValidatorExists(&delegation.Beneficiary) {
				validator := self.validators.GetValidator(&delegation.Beneficiary)
				if validator == nil {
					panic("registerValidator: validator does not exist")
				}
				prev_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

				validator.TotalStake.Add(validator.TotalStake, delegation.Value)
				self.validators.ModifyValidator(&delegation.Beneficiary, validator)

				state, state_k = self.state_get(delegation.Beneficiary[:], BlockToBytes(0))
				if state == nil {
					panic("registerValidator: broken state")
				}
				new_vote_count := voteCount(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
				if prev_vote_count != new_vote_count {
					self.eligible_vote_count -= prev_vote_count
					self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
				}
			} else {
				state, state_k = self.state_get(delegation.Beneficiary[:], BlockToBytes(0))
				if state != nil {
					panic("registerValidator: state already exists")
				}

				if !self.validators.CheckValidatorOwner(&common.ZeroAddress, &delegation.Beneficiary) {
					panic("registerValidator: owner already exists")
				}
				state = new(State)
				state.RewardsPer1Stake = bigutil.Big0
				self.validators.CreateValidator(delegator_address, &delegation.Beneficiary, 0, delegation.Value, args.Commission, args.Description, args.Endpoint)
				state.Count++
				new_vote_count := voteCount(delegation.Value, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
				if new_vote_count > 0 {
					self.eligible_vote_count = add64p(self.eligible_vote_count, new_vote_count)
				}
			}

			self.storage.SubBalance(delegator_address, delegation.Value)
			// Creates Delegation object in storage
			self.delegations.CreateDelegation(delegator_address, &delegation.Beneficiary, 0, delegation.Value)
			state.Count++
			self.state_put(&state_k, state)
			self.amount_delegated = bigutil.Add(self.amount_delegated, delegation.Value)
		}
	}
}

func (self *Contract) calculateRewardPer1Stake(rewardsPool *big.Int, stake *big.Int) *big.Int {
	return bigutil.Div(bigutil.Mul(rewardsPool, self.cfg.MaximumStake), stake)
}

func (self *Contract) calculateDelegatorReward(rewardPer1Stake *big.Int, stake *big.Int) *big.Int {
	return bigutil.Div(bigutil.Mul(rewardPer1Stake, stake), self.cfg.MaximumStake)
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
