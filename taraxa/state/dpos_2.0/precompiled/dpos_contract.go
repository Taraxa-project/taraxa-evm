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
	field_validators      = []byte{0}
	field_state           = []byte{1}
	field_delegations     = []byte{2}
	field_validators_info = []byte{3}

	field_eligible_count      = []byte{4}
	field_eligible_vote_count = []byte{5}
	field_amount_delegated    = []byte{6}
)

var Big10000 = new(big.Int).SetInt64(10000)

type Validator struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission *big.Int

	// Rewards accumulated
	RewardsPool *big.Int

	// Rewards accumulated
	CommissionRewardsPool *big.Int

	// Block number related to commission
	LastUpdated types.BlockNum
}

type ValidatorInfo struct {
	// Validators description
	Description string

	// Validators website endpoint
	Endpoint string
}

type Delegation struct {
	// Num of delegated tokens == delegator's stake
	Stake *big.Int

	// Block number related to rewards
	LastUpdated types.BlockNum
}

type State struct {
	RwardsPer1Stake *big.Int

	// number of references
	Count uint32
}

// This could be saved as one chunk? it will be rarely accesed
type ValidatorDelegation = map[common.Address]Delegation
type StateMap = map[*big.Int]State
type Contract struct {
	storage        StorageWrapper
	delayedStorage Reader
	Abi            abi.ABI

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

	// TODO: read delayedRequest from storage for blocks <last_commited_block_num+1, last_commited_block_num + 1 + delay>

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
	//Storage Update
	self.delayedStorage = readStorage
	//Handle withdrawals
	
	//Update values
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
		var args ValidatorArgs
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
		var args ValidatorArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse confirmUndelegate input args: ", err)
			return nil, err
		}

		return nil, self.confirmUndelegate(ctx, args)

	case "reDelegate":
		var args RedelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse reDelegate input args: ", err)
			return nil, err
		}

		return nil, self.redelegate(ctx, evm.GetBlock().Number, args)

	case "claimRewards":
		var args ValidatorArgs
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
		var args ValidatorArgs
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
		var args ValidatorArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}

		result := self.delayedStorage.GetValidatorEligibleVotesCount(&args.Validator)
		return method.Outputs.Pack(result)
	}

	return nil, nil
}

func (self *Contract) delegate(ctx vm.CallFrame, block types.BlockNum, args ValidatorArgs) error {
	validator, validator_k := self.validator_get(args.Validator[:])
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
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	delegation, delegation_k := self.delegation_get(args.Validator[:], ctx.Account.Address()[:])
	if delegation == nil {
		delegation = new(Delegation)
		ctx.Account.SubBalance(ctx.Value) // TODO how to get correct value?
		delegation.Stake = ctx.Value
		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value) // TODO how to get correct value?
	} else {
		// We need to claim rewards first
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
		ctx.Account.AddBalance(bigutil.Mul(reward, delegation.Stake))

		ctx.Account.SubBalance(ctx.Value) // TODO how to get correct value?
		delegation.Stake = bigutil.Add(delegation.Stake, ctx.Value)
		validator.TotalStake = bigutil.Add(validator.TotalStake, ctx.Value) // TODO how to get correct value?
	}
	delegation.LastUpdated = block
	state.Count++

	self.delegation_put(&delegation_k, delegation)
	self.state_put(&state_k, state)
	self.validator_put(&validator_k, validator)

	return nil
}

func (self *Contract) undelegate(ctx vm.CallFrame, block types.BlockNum, args UndelegateArgs) error {
	validator, validator_k := self.validator_get(args.Validator[:])
	if validator == nil {
		return ErrNonExistentValidator
	}

	delegation, delegation_k := self.delegation_get(args.Validator[:], ctx.Account.Address()[:])
	if delegation == nil {
		return ErrExistentValidator
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
	ctx.Account.AddBalance(bigutil.Mul(reward, delegation.Stake))

	ctx.Account.AddBalance(args.Amount) // TODO move it to wait queue
	delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
	validator.TotalStake = bigutil.Sub(validator.TotalStake, args.Amount)

	if delegation.Stake.Cmp(bigutil.Big0) == 0 {
		self.delegation_put(&delegation_k, nil)
	} else {
		delegation.LastUpdated = block
		state.Count++
		self.delegation_put(&delegation_k, delegation)
	}

	if validator.TotalStake.Cmp(bigutil.Big0) == 0 {
		self.validator_put(&validator_k, nil)
		self.validator_info_delete(args.Validator[:])
		self.state_put(&state_k, nil)
	} else {
		self.state_put(&state_k, state)
		self.validator_put(&validator_k, validator)
	}

	return nil
}

func (self *Contract) confirmUndelegate(ctx vm.CallFrame, args ValidatorArgs) error {
	return nil
}

func (self *Contract) redelegate(ctx vm.CallFrame, block types.BlockNum, args RedelegateArgs) error {
	validator_from, validator_from_k := self.validator_get(args.Validator_from[:])
	if validator_from == nil {
		return ErrNonExistentValidator
	}

	validator_to, validator_to_k := self.validator_get(args.Validator_to[:])
	if validator_to == nil {
		return ErrNonExistentValidator
	}
	//First we undelegate
	{
		delegation, delegation_k := self.delegation_get(args.Validator_from[:], ctx.Account.Address()[:])
		if delegation == nil {
			return ErrExistentValidator
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
		ctx.Account.AddBalance(bigutil.Mul(reward, delegation.Stake))

		delegation.Stake = bigutil.Sub(delegation.Stake, args.Amount)
		validator_from.TotalStake = bigutil.Sub(validator_from.TotalStake, args.Amount)

		if delegation.Stake.Cmp(bigutil.Big0) == 0 {
			self.delegation_put(&delegation_k, nil)
		} else {
			delegation.LastUpdated = block
			state.Count++
			self.delegation_put(&delegation_k, delegation)
		}

		if validator_from.TotalStake.Cmp(bigutil.Big0) == 0 {
			self.validator_put(&validator_from_k, nil)
			self.validator_info_delete(args.Validator_from[:])
			self.state_put(&state_k, nil)
		} else {
			self.state_put(&state_k, state)
			self.validator_put(&validator_from_k, validator_from)
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

		delegation, delegation_k := self.delegation_get(args.Validator_to[:], ctx.Account.Address()[:])
		if delegation == nil {
			delegation = new(Delegation)
			delegation.Stake = args.Amount
			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake,args.Amount)
		} else {
			// We need to claim rewards first
			old_state := self.state_get_and_decrement(args.Validator_to[:], BlockToBytes(delegation.LastUpdated))
			if old_state == nil {
				return ErrBrokenState
			}
			reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
			ctx.Account.AddBalance(bigutil.Mul(reward, delegation.Stake))
			delegation.Stake = bigutil.Add(delegation.Stake, args.Amount)
			validator_to.TotalStake = bigutil.Add(validator_to.TotalStake, args.Amount)
		}
		delegation.LastUpdated = block
		state.Count++

		self.delegation_put(&delegation_k, delegation)
		self.state_put(&state_k, state)
		self.validator_put(&validator_to_k, validator_to)
	}
	return nil
}

func (self *Contract) claimRewards(ctx vm.CallFrame, block types.BlockNum, args ValidatorArgs) error {
	delegation, delegation_k := self.delegation_get(args.Validator[:], ctx.Account.Address()[:])
	if delegation == nil {
		return ErrNonExistentDelegator
	}

	state, state_k := self.state_get(args.Validator[:], BlockToBytes(block))
	if state == nil {
		validator, validator_k := self.validator_get(args.Validator[:])
		if validator == nil {
			return ErrNonExistentValidator
		}
		old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
		self.validator_put(&validator_k, validator)
	}

	old_state := self.state_get_and_decrement(args.Validator[:], BlockToBytes(delegation.LastUpdated))
	if old_state == nil {
		return ErrBrokenState
	}
	reward := bigutil.Sub(state.RwardsPer1Stake, old_state.RwardsPer1Stake)
	ctx.Account.AddBalance(bigutil.Mul(reward, delegation.Stake))
	delegation.LastUpdated = block
	state.Count++

	self.delegation_put(&delegation_k, delegation)
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, block types.BlockNum) error {
	validator, validator_k := self.validator_get(ctx.Account.Address()[:])
	if validator == nil {
		return ErrNonExistentValidator
	}

	state, state_k := self.state_get(ctx.Account.Address()[:], BlockToBytes(block))
	if state == nil {
		old_state := self.state_get_and_decrement(ctx.Account.Address()[:], BlockToBytes(validator.LastUpdated))
		if old_state == nil {
			return ErrBrokenState
		}
		state = new(State)
		state.RwardsPer1Stake = bigutil.Add(old_state.RwardsPer1Stake, bigutil.Div(validator.RewardsPool, validator.TotalStake))
		validator.RewardsPool = bigutil.Big0
		validator.LastUpdated = block
		state.Count++
	}

	ctx.Account.AddBalance(validator.CommissionRewardsPool)
	validator.CommissionRewardsPool = bigutil.Big0

	self.validator_put(&validator_k, validator)
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) registerValidator(ctx vm.CallFrame, block types.BlockNum, args RegisterValidatorArgs) error {
	validator, validator_k := self.validator_get(ctx.Account.Address()[:])
	if validator != nil {
		return ErrExistentValidator
	}

	validator_info, validator_info_k := self.validator_info_get(ctx.Account.Address()[:])
	if validator_info != nil {
		return ErrExistentValidator
	}

	delegation, delegation_k := self.delegation_get(ctx.Account.Address()[:], ctx.Account.Address()[:])
	if delegation != nil {
		return ErrExistentValidator
	}

	state, state_k := self.state_get(ctx.Account.Address()[:], BlockToBytes(block))
	if state != nil {
		return ErrBrokenState
	}

	ctx.Account.SubBalance(ctx.Value) // TODO how to get correct value?

	state = new(State)
	state.RwardsPer1Stake = bigutil.Big0

	validator = new(Validator)
	validator.CommissionRewardsPool = bigutil.Big0
	validator.RewardsPool = bigutil.Big0
	validator.Commission = new(big.Int).SetUint64(args.Commission)
	validator.TotalStake = ctx.Value // TODO how to get correct value?
	validator.LastUpdated = block
	state.Count++

	delegation = new(Delegation)
	delegation.Stake = ctx.Value // TODO how to get correct value?
	delegation.LastUpdated = block
	state.Count++

	validator_info.Description = args.Description
	validator_info.Endpoint = args.Endpoint

	self.validator_info_put(&validator_info_k, validator_info)
	self.validator_put(&validator_k, validator)
	self.delegation_put(&delegation_k, delegation)
	self.state_put(&state_k, state)

	return nil
}

func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
	validator_info, validator_info_k := self.validator_info_get(ctx.Account.Address()[:])
	if validator_info == nil {
		return ErrNonExistentValidator
	}

	validator_info.Description = args.Description
	validator_info.Endpoint = args.Endpoint
	self.validator_info_put(&validator_info_k, validator_info)

	return nil
}

func (self *Contract) setCommission(ctx vm.CallFrame, args SetCommissionArgs) error {
	validator, validator_k := self.validator_get(ctx.Account.Address()[:])
	if validator == nil {
		return ErrNonExistentValidator
	}

	validator.Commission = new(big.Int).SetUint64(args.Commission)
	self.validator_put(&validator_k, validator)

	return nil
}

func (self *Contract) update_rewards(validator_addr *common.Address, reward *big.Int) {
	validator, validator_k := self.validator_get(validator_addr[:])
	if validator != nil {
		commission := bigutil.Mul(bigutil.Div(reward, Big10000), validator.Commission )
		validator.CommissionRewardsPool = bigutil.Add(validator.CommissionRewardsPool, commission)
		validator.RewardsPool = bigutil.Add(validator.RewardsPool, bigutil.Sub(reward, commission))
		self.validator_put(&validator_k, validator)
	}
}

func (self *Contract) validator_get(validator_addr []byte) (validator *Validator, key common.Hash) {
	key = stor_k_2(field_validators, validator_addr)
	self.storage.Get(&key, func(bytes []byte) {
		validator = new(Validator)
		rlp.MustDecodeBytes(bytes, validator)
	})
	return
}

func (self *Contract) validator_put(key *common.Hash, validator *Validator) {
	if validator != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(validator))
	} else {
		self.storage.Put(key, nil)
	}
}

func (self *Contract) validator_info_get(validator_addr []byte) (validator_info *ValidatorInfo, key common.Hash) {
	key = stor_k_2(field_validators_info, validator_addr)
	self.storage.Get(&key, func(bytes []byte) {
		validator_info = new(ValidatorInfo)
		rlp.MustDecodeBytes(bytes, validator_info)
	})
	return
}

func (self *Contract) validator_info_put(key *common.Hash, validator_info *ValidatorInfo) {
	if validator_info != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(validator_info))
	} else {
		self.storage.Put(key, nil)
	}
}

func (self *Contract) validator_info_delete(validator_addr []byte)  {
	key := stor_k_1(field_validators_info, validator_addr)
	self.storage.Put(key, nil)
	return
}

func (self *Contract) delegation_get(validator_addr, delegator_addr []byte) (delegation *Delegation, key common.Hash) {
	key = stor_k_2(field_delegations, validator_addr, delegator_addr)
	self.storage.Get(&key, func(bytes []byte) {
		delegation = new(Delegation)
		rlp.MustDecodeBytes(bytes, delegation)
	})
	return
}

func (self *Contract) delegation_put(key *common.Hash, delegation *Delegation) {
	if delegation != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(delegation))
	} else {
		self.storage.Put(key, nil)
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
