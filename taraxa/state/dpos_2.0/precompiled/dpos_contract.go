package dpos_2

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

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
var ErrNonReadyUndelegation = util.ErrorString("Undelegation is not yet ready to be withdrawn")
var ErrExistentValidator = util.ErrorString("Validator already exist")
var ErrBrokenState = util.ErrorString("Fatal error state is broken")
var ErrNonExistentDelegator = util.ErrorString("Delegator does not exist")
var ErrValidatorsMaxStakeExceeded = util.ErrorString("Validator's max stake exceeded")
var ErrInsufficientDelegation = util.ErrorString("Insufficient delegation")

var ErrTransferAmountIsZero = util.ErrorString("transfer amount is zero")
var ErrWithdrawalExceedsDeposit = util.ErrorString("withdrawal exceeds prior deposit value")
var ErrInsufficientBalanceForDeposits = util.ErrorString("insufficient balance for the deposits")
var ErrCallIsNotToplevel = util.ErrorString("only top-level calls are allowed")
var ErrNoTransfers = util.ErrorString("no transfers")
var ErrCallValueNonzero = util.ErrorString("call value must be zero")
var ErrDuplicateBeneficiary = util.ErrorString("duplicate beneficiary")

// Contract storage fields keys
var (
	field_validators  	= []byte{0}
	field_state       	= []byte{1}
	field_delegations 	= []byte{2}
	field_undelegations = []byte{3}

	field_eligible_count      = []byte{4}
	field_eligible_vote_count = []byte{5}
	field_amount_delegated    = []byte{6}
)

var Big10000 = new(big.Int).SetInt64(10000)

// Maximum number of validators per batch returned by getValidators call
const GetValidatorsMaxCount = 50

// Maximum number of validators per batch returned by getDelegatorDelegations call
const GetDelegatorDelegationsMaxCount = 50

// Undelegation delay 300k blocks ~20 day if time is 6s (this should be part of config)
const UndelegationDelay = 300000

type State struct {
	RwardsPer1Stake *big.Int

	// number of references
	Count uint32
}

type Contract struct {
	storage        StorageWrapper
	delayedStorage Reader
	Abi            abi.ABI

	validators    Validators
	delegations   Delegations
	undelegations Undelegations

	eligible_count_orig      uint64
	eligible_count           uint64
	eligible_vote_count_orig uint64
	eligible_vote_count      uint64
	amount_delegated_orig    *big.Int
	amount_delegated         *big.Int
	lazy_init_done           bool
}

func (self *Contract) Init(storage Storage, readStorage Reader) *Contract {
	self.storage.Init(storage)
	self.delayedStorage = readStorage
	dpos_abi, err := ioutil.ReadFile("DposInterface.abi")
	if err != nil {
		panic("Unable to load dpos contract interface abi: " + err.Error())
	}
	self.Abi, _ = abi.JSON(strings.NewReader(string(dpos_abi)))

	self.validators.Init(self.storage, field_validators)
	self.delegations.Init(self.storage, field_delegations)
	self.undelegations.Init(self.storage, field_undelegations)

	return self
}

func (self *Contract) UpdateStorage(readStorage Reader) {
	self.delayedStorage = readStorage
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
	self.storage.Get(stor_k_1(field_eligible_count), func(bytes []byte) {
		self.eligible_count_orig = bin.DEC_b_endian_compact_64(bytes)
	})
	self.eligible_count = self.eligible_count_orig
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

func (self *Contract) EndBlockCall(readStorage Reader, blk_n types.BlockNum) {
	defer self.storage.ClearCache()
	// Storage Update
	self.delayedStorage = readStorage

	// Update values
	if self.eligible_count_orig != self.eligible_count {
		self.storage.Put(stor_k_1(field_eligible_count), bin.ENC_b_endian_compact_64_1(self.eligible_count))
		self.eligible_count_orig = self.eligible_count
	}
	if self.eligible_vote_count_orig != self.eligible_vote_count {
		self.storage.Put(stor_k_1(field_eligible_vote_count), bin.ENC_b_endian_compact_64_1(self.eligible_vote_count))
		self.eligible_vote_count_orig = self.eligible_vote_count
	}
	if self.amount_delegated_orig.Cmp(self.amount_delegated) != 0 {
		self.storage.Put(stor_k_1(field_amount_delegated), self.amount_delegated.Bytes())
		self.amount_delegated_orig = self.amount_delegated
	}
}

func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if ctx.Value.Sign() != 0 {
		return nil, ErrCallValueNonzero
	}

	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}

	method, err := self.Abi.MethodById(ctx.Input)
	if err != nil {
		fmt.Println("Unknown method: ", err)
		return nil, nil
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
		return nil, self.claimCommissionRewards(ctx, evm.GetBlock().Number)

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
		result := self.delayedStorage.IsValidatorEligible(&args.Validator)
		return method.Outputs.Pack(result)

	case "getTotalEligibleValidatorsCount":
		result := self.delayedStorage.GetTotalEligibleValidatorsCount()
		return method.Outputs.Pack(result)

	case "getTotalEligibleVotesCount":
		result := self.delayedStorage.GetTotalEligibleVotesCount()
		return method.Outputs.Pack(result)

	case "getValidatorEligibleVotesCount":
		var args ValidatorAddress
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}

		result := self.delayedStorage.GetValidatorEligibleVotesCount(&args.Validator)
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
	}

	return nil, nil
}

func (self *Contract) delegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	validator := self.validators.GetValidator(&args.Validator)
	if validator == nil {
		return ErrNonExistentValidator
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		// TODO: question: how can we erase validator's RewardsPool during delegation of single delegator ???
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
		if old_state == nil {
			return ErrBrokenState
		}
		reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))

		ctx.Account.SubBalance(ctx.Value)

		delegation.Stake = bigutil.Add(delegation.Stake, ctx.Value)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value)
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

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	// We need to claim rewards first
	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	if old_state == nil {
		return ErrBrokenState
	}

	reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
	// Reward needs to be add to callers accounts as only stake is locked
	ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))

	// Creating undelegation request
	self.undelegations.CreateUndelegation(ctx.CallerAccount.Address(), &args.Validator, block + UndelegationDelay, args.Amount)
	delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
	validator.TotalStake = bigutil.Sub(validator.TotalStake, args.Amount)

	if delegation.Stake.Cmp(bigutil.Big0) == 0 {
		self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.Validator)
	} else {
		delegation.LastUpdated = block
		state.Count++
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)
	}

	// TODO: cant do this, RewardsPool as well as CommissionRewardsPool must be == 0 too
	if validator.TotalStake.Cmp(bigutil.Big0) == 0 {
		self.validators.DeleteValidator(&args.Validator)
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
		return ErrNonReadyUndelegation
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
	undelegation := self.undelegations.GetUndelegation(ctx.CallerAccount.Address(), &args.Validator)
	self.undelegations.RemoveUndelegation(ctx.CallerAccount.Address(), &args.Validator)

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		// TODO: question: how can we erase validator's RewardsPool during delegation of single delegator ???
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
		if old_state == nil {
			return ErrBrokenState
		}
		reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))

		delegation.Stake = bigutil.Add(delegation.Stake, undelegation.Amount)
		delegation.LastUpdated = block
		self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

		validator.TotalStake = bigutil.Add(validator.TotalStake, undelegation.Amount)
	}

	state.Count++
	self.state_put(&state_k, state)
	self.validators.ModifyValidator(&args.Validator, validator)
	return nil
}

func (self *Contract) redelegate(ctx vm.CallFrame, block types.BlockNum, args RedelegateArgs) error {
	//validator_from, validator_from_k := self.validator_get(args.Validator_from[:])
	validator_from := self.validators.GetValidator(&args.Validator_from)
	if validator_from == nil {
		return ErrNonExistentValidator
	}

	validator_to := self.validators.GetValidator(&args.Validator_to)
	if validator_to == nil {
		return ErrNonExistentValidator
	}
	//First we undelegate
	{
		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator_from)
		if delegation == nil {
			return ErrNonExistentDelegation
		}

		if delegation.Stake.Cmp(args.Amount) == -1 {
			return ErrInsufficientDelegation
		}

		state, state_k := self.state_get(args.Validator_from[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.Validator_from[:], BlockToBytes(validator_from.LastUpdated))
			if old_state == nil {
				return ErrBrokenState
			}
			state = new(State)
			state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator_from.RewardsPool, validator_from.TotalStake))
			validator_from.RewardsPool = bigutil.Big0
			validator_from.LastUpdated = block
			state.Count++
		}
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator_from[:], BlockToBytes(delegation.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
		ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))

		delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
		validator_from.TotalStake = bigutil.Sub(validator_from.TotalStake, args.Amount)

		if delegation.Stake.Cmp(bigutil.Big0) == 0 {
			self.delegations.RemoveDelegation(ctx.CallerAccount.Address(), &args.Validator_from)
		} else {
			delegation.LastUpdated = block
			state.Count++
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator_from, delegation)
		}

		// TODO: cant do this, RewardsPool as well as CommissionRewardsPool must be == 0 too
		if validator_from.TotalStake.Cmp(bigutil.Big0) == 0 {
			self.validators.DeleteValidator(&args.Validator_from)
			self.state_put(&state_k, nil)
		} else {
			self.state_put(&state_k, state)
			self.validators.ModifyValidator(&args.Validator_from, validator_from)
		}
	}

	// Now we delegate
	{
		state, state_k := self.state_get(args.Validator_to[:], BlockToBytes(block))
		if state == nil {
			old_state := self.state_get_and_decrement(args.Validator_to[:], BlockToBytes(validator_to.LastUpdated))
			if old_state == nil {
				return ErrBrokenState
			}
			state = new(State)
			state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator_to.RewardsPool, validator_to.TotalStake))
			validator_to.RewardsPool = bigutil.Big0
			validator_to.LastUpdated = block
			state.Count++
		}

		delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator_to)
		if delegation == nil {
			self.delegations.CreateDelegation(ctx.CallerAccount.Address(), &args.Validator_to, block, args.Amount)
			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		} else {
			// We need to claim rewards first
			old_state := self.state_get_and_decrement(args.Validator_to[:], BlockToBytes(delegation.LastUpdated))
			if old_state == nil {
				return ErrBrokenState
			}
			reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
			ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))

			delegation.Stake = bigutil.Add(delegation.Stake, args.Amount)
			delegation.LastUpdated = block
			self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator_to, delegation)

			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		}

		state.Count++

		self.state_put(&state_k, state)
		self.validators.ModifyValidator(&args.Validator_to, validator_to)
	}
	return nil
}

func (self *Contract) claimRewards(ctx vm.CallFrame, block types.BlockNum, args ValidatorAddress) error {
	delegation := self.delegations.GetDelegation(ctx.CallerAccount.Address(), &args.Validator)
	if delegation == nil {
		return ErrNonExistentDelegator
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		validator := self.validators.GetValidator(&args.Validator)
		if validator == nil {
			return ErrNonExistentValidator
		}
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		// TODO: question: how can we reset validator's rewards pool after singl delegator claim rewards ?
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
		self.validators.ModifyValidator(&args.Validator, validator)
	}

	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	if old_state == nil {
		return ErrBrokenState
	}

	reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
	// TODO: question: how is it possible that in case state == nil, we give delegator some rewards but we dont adjust validator's rewards pool ???
	ctx.CallerAccount.AddBalance(bigutil.Mul(reward, delegation.Stake))
	delegation.LastUpdated = block
	self.delegations.ModifyDelegation(ctx.CallerAccount.Address(), &args.Validator, delegation)

	state.Count++
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, block types.BlockNum) error {
	validator_address := ctx.CallerAccount.Address()
	validator := self.validators.GetValidator(validator_address)
	if validator == nil {
		return ErrNonExistentValidator
	}

	state, state_k := self.state_get(ctx.CallerAccount.Address()[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(ctx.CallerAccount.Address()[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	ctx.CallerAccount.AddBalance(validator.CommissionRewardsPool)
	validator.CommissionRewardsPool = bigutil.Big0

	self.validators.ModifyValidator(validator_address, validator)
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) registerValidator(ctx vm.CallFrame, block types.BlockNum, args RegisterValidatorArgs) error {
	validator_address := ctx.CallerAccount.Address()

	if self.validators.ValidatorExists(validator_address) {
		return ErrExistentValidator
	}

	delegation := self.delegations.GetDelegation(validator_address, validator_address)
	if delegation != nil {
		// This could happen only due some serious logic bug
		panic("registerValidator: delegation already exists")
	}

	state, state_k := self.state_get(validator_address[:], BlockToBytes(block))
	if state != nil {
		return ErrBrokenState
	}

	// TODO: limit size of description & endpoint - should be very small

	ctx.Account.SubBalance(ctx.Value) // TODO how to get correct value?

	state = new(State)
	state.RwardsPer1Stake = bigutil.Big0

	// Creates validator related objects in storage
	self.validators.CreateValidator(validator_address, block, ctx.Value, args.Commission, args.Description, args.Endpoint)
	state.Count++

	// Creates Delegation object in storage
	self.delegations.CreateDelegation(validator_address, validator_address, block, ctx.Value)
	state.Count++

	self.state_put(&state_k, state)
	return nil
}

func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
	validator_address := ctx.CallerAccount.Address()

	validator_info := self.validators.GetValidatorInfo(validator_address)
	if validator_info == nil {
		return ErrNonExistentValidator
	}

	// TODO: limit max size of endpoint & description

	validator_info.Description = args.Description
	validator_info.Endpoint = args.Endpoint

	self.validators.ModifyValidatorInfo(validator_address, validator_info)

	return nil
}

func (self *Contract) setCommission(ctx vm.CallFrame, args SetCommissionArgs) error {
	validator_address := ctx.CallerAccount.Address()
	validator := self.validators.GetValidator(validator_address)
	if validator == nil {
		return ErrNonExistentValidator
	}

	validator.Commission = args.Commission
	self.validators.ModifyValidator(validator_address, validator)

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
		current_reward := bigutil.Add(state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		reward := bigutil.Sub(current_reward, old_state.RwardsPer1Stake)
		////

		delegation_data.Delegation.Rewards = bigutil.Mul(reward, delegation.Stake)
		result.Delegations = append(result.Delegations, delegation_data)
	}

	result.End = end
	return
}

func (self *Contract) update_rewards(validator_address *common.Address, reward *big.Int) {
	validator := self.validators.GetValidator(validator_address)

	// TODO: situation when validator == nil should never happen, how to handle it ?
	if validator != nil {
		commission := bigutil.Mul(bigutil.Div(reward, Big10000), big.NewInt(int64(validator.Commission)))
		validator.CommissionRewardsPool = bigutil.Add(validator.CommissionRewardsPool, commission)
		validator.RewardsPool = bigutil.Add(validator.RewardsPool, bigutil.Sub(reward, commission))
		self.validators.ModifyValidator(validator_address, validator)
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

func BlockToBytes(number types.BlockNum) []byte {
	big := new(big.Int)
	big.SetUint64(number)
	return big.Bytes()
}