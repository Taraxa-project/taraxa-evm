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
)

var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000FE"))

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

// Contract storage fields keys
var (
	field_validators      = []byte{0}
	field_validator2owner = []byte{1}
	field_state           = []byte{2}
	field_delegations     = []byte{3}
	field_undelegations   = []byte{4}

	field_eligible_vote_count = []byte{5}
	field_amount_delegated    = []byte{6}
)

var Big10000 = new(big.Int).SetInt64(10000)

// Maximum number of validators per batch returned by getValidators call
const GetValidatorsMaxCount = 50

// Maximum number of validators per batch returned by getDelegatorDelegations call
const GetDelegatorDelegationsMaxCount = 50

type State struct {
	RewardsPer1Stake *big.Int

	// number of references
	Count uint32
}

type Contract struct {
	cfg            Config
	storage        StorageWrapper
	delayedStorage Reader
	Abi            abi.ABI

	validators    Validators
	delegations   Delegations
	undelegations Undelegations

	eligible_vote_count_orig uint64
	eligible_vote_count      uint64
	amount_delegated_orig    *big.Int
	amount_delegated         *big.Int
	lazy_init_done           bool
}

func (self *Contract) Init(cfg Config, storage Storage, readStorage Reader) *Contract {
	self.cfg = cfg
	self.storage.Init(storage)
	self.delayedStorage = readStorage
	self.Abi, _ = abi.JSON(strings.NewReader(TaraxaDposClientMetaData))

	self.validators.Init(&self.storage, field_validators)
	self.delegations.Init(&self.storage, field_delegations)
	self.undelegations.Init(&self.storage, field_undelegations)

	return self
}

func (self *Contract) UpdateStorage(readStorage Reader) {
	self.delayedStorage = readStorage
}

func (self *Contract) UpdateConfig(cfg Config) {
	self.cfg = cfg
}

func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	// TODO: based on method being called, calculate the gas somehow. Doing it based on the length of input is totally useless
	return uint64(len(ctx.Input)) * 20
}

func (self *Contract) lazy_init() {
	if self.lazy_init_done {
		return
	}
	self.lazy_init_done = true
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

func (self *Contract) BeginBlockCall(rewards map[common.Address]*big.Int) {
	for validator, reward := range rewards {
		self.update_rewards(&validator, reward)
	}
}

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

func (self *Contract) CommitCall(readStorage Reader) {
	defer self.storage.ClearCache()
	// Storage Update
	self.delayedStorage = readStorage
}

func (self *Contract) ApplyGenesis() error {
	self.lazy_init()

	for _, entry := range self.cfg.GenesisState {
		self.apply_genesis_entry(&entry.Benefactor, entry.Transfers)
	}

	self.EndBlockCall()
	self.storage.IncrementNonce(contract_address)
	return nil
}

func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}

	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		return nil, err
	}

	self.lazy_init()

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

func (self *Contract) delegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}
	prev_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		ctx.Account.SubBalance(ctx.Value)
		self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator, block, ctx.Value)
		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value)
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))

		ctx.Account.SubBalance(ctx.Value)

		delegation.Stake = bigutil.Add(delegation.Stake, ctx.Value)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value)
	}

	self.amount_delegated = bigutil.Add(self.amount_delegated, ctx.Value)
	new_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(&args.Validator, validator)
	return nil
}

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

	prev_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	// We need to claim rewards first
	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
	// Reward needs to be add to callers accounts as only stake is locked
	ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))

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
	new_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
	}

	if validator.TotalStake.Cmp(bigutil.Big0) == 0 && validator.CommissionRewardsPool.Cmp(bigutil.Big0) == 0 {
		self.validators.DeleteValidator(&args.Validator)
		self.set_validator_owner(nil, &args.Validator)
		self.state_put(&state_k, nil)
	} else {
		self.state_put(&state_k, state)
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	return nil
}

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

func (self *Contract) cancelUndelegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	if !self.undelegations.UndelegationExists(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrNonExistentUndelegation
	}
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}
	prev_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		state = new(State)
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
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
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))

		delegation.Stake = bigutil.Add(delegation.Stake, undelegation.Amount)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, undelegation.Amount)
	}
	self.amount_delegated = bigutil.Add(self.amount_delegated, undelegation.Amount)
	new_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	if prev_vote_count != new_vote_count {
		self.eligible_vote_count -= prev_vote_count
		self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(&args.Validator, validator)
	return nil
}

func (self *Contract) redelegate(ctx vm.CallFrame, block types.BlockNum, args RedelegateArgs) error {
	validator_from := self.validators.GetValidator(&args.ValidatorFrom)
	if validator_from == nil {
		return ErrNonExistentValidator
	}

	validator_to := self.validators.GetValidator(&args.ValidatorTo)
	if validator_to == nil {
		return ErrNonExistentValidator
	}

	prev_vote_count_from := vote_count(validator_from.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	prev_vote_count_to := vote_count(validator_to.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
	//First we undelegate
	{
		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.ValidatorFrom)
		if delegation == nil {
			return ErrNonExistentDelegation
		}

		if delegation.Stake.Cmp(args.Amount) == -1 {
			return ErrInsufficientDelegation
		}

		state, state_k := self.state_get(args.ValidatorFrom[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(validator_from.LastUpdated))
			state = new(State)
			state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator_from.RewardsPool, validator_from.TotalStake))
			validator_from.RewardsPool = bigutil.Big0
			validator_from.LastUpdated = block
			state.Count++
		}
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.ValidatorFrom[:], BlockToBytes(delegation.LastUpdated))
		reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))

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
			self.set_validator_owner(nil, &args.ValidatorFrom)
			self.state_put(&state_k, nil)
		} else {
			self.state_put(&state_k, state)
			self.validators.ModifyValidator(&args.ValidatorFrom, validator_from)
		}

		new_vote_count := vote_count(validator_from.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if prev_vote_count_from != new_vote_count {
			self.eligible_vote_count -= prev_vote_count_from
			self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
		}

	}

	// Now we delegate
	{
		state, state_k := self.state_get(args.ValidatorTo[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.ValidatorTo[:], BlockToBytes(validator_to.LastUpdated))
			state = new(State)
			state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator_to.RewardsPool, validator_to.TotalStake))
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
			ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))

			delegation.Stake = bigutil.Add(delegation.Stake, args.Amount)
			delegation.LastUpdated = block
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.ValidatorTo, delegation)

			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		}

		new_vote_count := vote_count(validator_to.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if prev_vote_count_to != new_vote_count {
			self.eligible_vote_count -= prev_vote_count_to
			self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
		}

		state.Count++
		self.state_put(&state_k, state)
		self.validators.ModifyValidator(&args.ValidatorTo, validator_to)
	}
	return nil
}

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
		state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	reward_per_stake := bigutil.Sub(state.RewardsPer1Stake, old_state.RewardsPer1Stake)
	ctx.CallerAccount.AddBalance(bigutil.Mul(reward_per_stake, delegation.Stake))
	delegation.LastUpdated = block
	self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

	state.Count++
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	if !self.check_validator_owner(ctx.CallerAccount.Address(), &args.Validator) {
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
		self.set_validator_owner(nil, &args.Validator)
		self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
	} else {
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	return nil
}

func (self *Contract) registerValidator(ctx vm.CallFrame, block types.BlockNum, args RegisterValidatorArgs) error {
	// make sure the public key is a valid one
	pubKey, err := crypto.Ecrecover(args.Validator.Hash().Bytes(), args.Proof)
	// the first byte of pubkey is bitcoin heritage
	if err != nil {
		return err
	}

	if common.BytesToAddress(keccak256.Hash(pubKey[1:])[12:]) != args.Validator {
		return ErrWrongProof
	}
	owner_address := ctx.CallerAccount.Address()
	if !self.check_validator_owner(&common.ZeroAddress, &args.Validator) {
		return ErrExistentValidator
	}

	if self.validators.ValidatorExists(&args.Validator) {
		return ErrExistentValidator
	}

	delegation := self.delegations.GetDelegation(owner_address, &args.Validator)
	if delegation != nil {
		// This could happen only due some serious logic bug
		panic("registerValidator: delegation already exists")
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state != nil {
		return ErrBrokenState
	}

	self.set_validator_owner(owner_address, &args.Validator)

	// TODO: limit size of description & endpoint - should be very small

	ctx.Account.SubBalance(ctx.Value)

	state = new(State)
	state.RewardsPer1Stake = bigutil.Big0

	// Creates validator related objects in storage
	self.validators.CreateValidator(&args.Validator, block, ctx.Value, args.Commission, args.Description, args.Endpoint)
	state.Count++

	if ctx.Value.Cmp(bigutil.Big0) == 1 {
		new_vote_count := vote_count(ctx.Value, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
		if new_vote_count > 0 {
			self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
		}
		self.amount_delegated = bigutil.Add(self.amount_delegated, ctx.Value)
		// Creates Delegation object in storage
		self.delegations.CreateDelegation(owner_address, &args.Validator, block, ctx.Value)
		state.Count++
	}
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
	if !self.check_validator_owner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}

	validator_info := self.validators.GetValidatorInfo(&args.Validator)
	if validator_info == nil {
		return ErrNonExistentValidator
	}

	// TODO: limit max size of endpoint & description

	validator_info.Description = args.Description
	validator_info.Endpoint = args.Endpoint

	self.validators.ModifyValidatorInfo(&args.Validator, validator_info)

	return nil
}

func (self *Contract) setCommission(ctx vm.CallFrame, args SetCommissionArgs) error {
	if !self.check_validator_owner(ctx.CallerAccount.Address(), &args.Validator) {
		return ErrWrongOwnerAcc
	}
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	validator.Commission = args.Commission
	self.validators.ModifyValidator(&args.Validator, validator)

	return nil
}

// TODO: measure performance of this call - if it is too bad -> decrease GetValidatorsMaxCount constant
func (self *Contract) getValidators(args GetValidatorsArgs) (result GetValidatorsRet) {
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

// TODO: measure performance of this call - if it is too bad -> decrease GetValidatorsMaxCount constant
// TODO: this will be super expensice call probably
func (self *Contract) getDelegatorDelegations(args GetDelegatorDelegationsArgs) (result GetDelegatorDelegationRet) {
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
		current_reward := bigutil.Add(state.RewardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		reward := bigutil.Sub(current_reward, old_state.RewardsPer1Stake)
		////

		delegation_data.Delegation.Rewards = bigutil.Mul(reward, delegation.Stake)
		result.Delegations = append(result.Delegations, delegation_data)
	}

	result.End = end
	return
}

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

func (self *Contract) update_rewards(validator_address *common.Address, reward *big.Int) {
	validator := self.validators.GetValidator(validator_address)

	// TODO: situation when validator == nil should never happen, how to handle it ?
	if validator != nil {
		commission := bigutil.Div(bigutil.Mul(reward, big.NewInt(int64(validator.Commission))), Big10000)
		validator.CommissionRewardsPool = bigutil.Add(validator.CommissionRewardsPool, commission)
		validator.RewardsPool = bigutil.Add(validator.RewardsPool, bigutil.Sub(reward, commission))
		self.validators.ModifyValidator(validator_address, validator)
	} else {
		panic("update_rewards - non exexistent validator")
	}
}

func (self *Contract) state_get(validator_addr, block []byte) (state *State, key common.Hash) {
	key = stor_k_2(field_state, validator_addr, block)
	self.storage.Get(&key, func(bytes []byte) {
		state = new(State)
		rlp.MustDecodeBytes(bytes, state)
	})
	return
}

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

func (self *Contract) state_put(key *common.Hash, state *State) {
	if state != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(state))
	} else {
		self.storage.Put(key, nil)
	}
}

func (self *Contract) check_validator_owner(owner, validator *common.Address) bool {
	key := stor_k_1(field_validator2owner, validator[:])
	var saved_addr common.Address
	self.storage.Get(key, func(bytes []byte) {
		saved_addr = common.BytesToAddress(bytes)
	})
	return *owner == saved_addr
}

func (self *Contract) set_validator_owner(owner, validator *common.Address) {
	key := stor_k_1(field_validator2owner, validator[:])
	if owner != nil {
		self.storage.Put(key, owner.Bytes())
	} else {
		self.storage.Put(key, nil)
	}
}

func (self *Contract) apply_genesis_entry(delegator_address *common.Address, transfers []GenesisTransfer) {
	// TODO fill them?
	var args RegisterValidatorArgs

	for _, delegation := range transfers {
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
				prev_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)

				validator.TotalStake.Add(validator.TotalStake, delegation.Value)
				self.validators.ModifyValidator(&delegation.Beneficiary, validator)

				state, state_k = self.state_get(delegation.Beneficiary[:], BlockToBytes(0))
				if state == nil {
					panic("registerValidator: broken state")
				}
				new_vote_count := vote_count(validator.TotalStake, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
				if prev_vote_count != new_vote_count {
					self.eligible_vote_count -= prev_vote_count
					self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
				}
			} else {
				state, state_k = self.state_get(delegation.Beneficiary[:], BlockToBytes(0))
				if state != nil {
					panic("registerValidator: state already exists")
				}

				if !self.check_validator_owner(&common.ZeroAddress, &delegation.Beneficiary) {
					panic("registerValidator: owner already exists")
				}
				self.set_validator_owner(delegator_address, &delegation.Beneficiary)

				state = new(State)
				state.RewardsPer1Stake = bigutil.Big0
				self.validators.CreateValidator(&delegation.Beneficiary, 0, delegation.Value, args.Commission, args.Description, args.Endpoint)
				state.Count++
				new_vote_count := vote_count(delegation.Value, self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
				if new_vote_count > 0 {
					self.eligible_vote_count = Add64p(self.eligible_vote_count, new_vote_count)
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

func BlockToBytes(number types.BlockNum) []byte {
	big := new(big.Int)
	big.SetUint64(number)
	return big.Bytes()
}

func vote_count(staking_balance, eligibility_threshold, vote_eligibility_balance_step *big.Int) uint64 {
	tmp := big.NewInt(0)
	if staking_balance.Cmp(eligibility_threshold) >= 0 {
		tmp.Div(staking_balance, vote_eligibility_balance_step)
	}
	asserts.Holds(tmp.IsUint64())
	return tmp.Uint64()
}

func Add64p(a, b uint64) uint64 {
	c := a + b
	if c < a || c < b {
		panic("addition overflow " + strconv.FormatUint(a, 10) + " " + strconv.FormatUint(b, 10))
	}
	return c
}
