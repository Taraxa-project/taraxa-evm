package dpos_2

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000FE"))

var ErrInsufficientBalance = util.ErrorString("Insufficient balance")
var ErrNonExistentValidator = util.ErrorString("Validator does not exist")
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

// Validator basic info
type ValidatorBasicInfo struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission *big.Int

	// Rewards accumulated from delegators rewards based on commission
	CommissionRewards *big.Int

	// Short description
	// TODO: optional - maybe we dont want this ?
	Description string

	// Validator's website url
	// TODO: optional - maybe we dont want this ?
	Endpoint string
}

// Validator info
type ValidatorInfo struct {
	// Validtor basic info
	BasicInfo ValidatorBasicInfo

	// List of validator's delegators
	Delegators map[common.Address]*DelegatorInfo
}

type DelegatorInfo struct {
	// Num of delegated tokens == delegator's stake
	Stake *big.Int

	// UnlockedStake == unlocked(undelegated) tokens that can be withdrawn now
	// TODO: in case we will send unlocked tokens to the delegator's balance automatically, we dont need this field
	UnlockedStake *big.Int

	// Accumulated rewards
	Rewards *big.Int

	// Undelegate request
	// TODO: rethink implementation of undelegations
	UndelegateRequests []*UndelegateRequest
}

type UndelegateRequest struct {
	// Num of tokens that delegator wants to undelegate
	Amount *big.Int

	// Block number when this unstake request can be confirmed(act block num + locking period)
	EligibleBlockNum *big.Int
}

// Delegator's validators info
type DelegatorValidators struct {
	// List of validators addresses that delegator delegated to
	// Note: info about delegator's stake/reward, etc... is saved in ValidatorDelegators struct
	Validators map[common.Address]bool // instead of set
}

// Contract storage fields keys
var (
	field_validators_info 		= []byte{0}
	// field_validators_delegators = []byte{1}
	field_delegators_validators = []byte{1}
	field_delayed_requests      = []byte{2}
	field_eligible_count 		= []byte{3}
	field_eligible_vote_count	= []byte{4}
	field_amount_delegated 		= []byte{5}
)

type Contract struct {
	Storage StorageWrapper
	DelayedStorage Reader
	Abi     abi.ABI

	// Validadors basic info, e.g. total stake, commission, etc...
	ValidatorsInfo map[common.Address]*ValidatorInfo

	// Delegators list of their validators -> key = delegator address
	DelegatorsValidators map[common.Address]*DelegatorValidators
}

// TODO: mappings that are going to be in in memory as well as storage
// 1. validatorsInfo: address -> ValidatorInfo
// 2. validatorsDelegators: address -> ValidatorDelegators
// 3. delegatorValidators: address -> DelegatorValidators
// 4. delayedRequest: block_num -> []DelayedRequests				// delayed delegation & undelegation requests that will be processed at the end of block_num.
// 5. delayedUndelegations: block_num -> []DelayedRequests  // delayed undelegations
// 6. delegatorDelayedUndelegations: address -> []DelayedRequests		// delayed undelegations for specific user - it is needed to check if he can do another undelegation request due to inefficient remaining stake
//
// Notes:
// 4. are processed automatically in Commit() function
// 6. does not need to be saved in storage - just memory is ok

func (self *Contract) Init(storage Storage, readStorage Reader) *Contract {
	self.Storage.Init(storage)
	self.DelayedStorage = readStorage
	dpos_abi, err := ioutil.ReadFile("DposInterface.abi")
	if err != nil {
		panic("Unable to load dpos contract interface abi: " + err.Error())
	}
	self.Abi, _ = abi.JSON(strings.NewReader(string(dpos_abi)))

	// TODO: read delayedRequest from storage for blocks <last_commited_block_num+1, last_commited_block_num + 1 + delay>

	return self
}

func (self *Contract) UpdateStorage(readStorage Reader) {
	self.DelayedStorage = readStorage
}

func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	// TODO: based on method being called, calculate the gas somehow. Doing it based on the length of input is totally useless
	return uint64(len(ctx.Input)) * 20
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

	// First 4 bytes is method signature !!!!
	input := ctx.Input[4:]

	switch method.Name {
	case "delegate":
		var args DelegateArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse delegate input args: ", err)
			return nil, err
		}

		return nil, self.delegate(ctx, args)

	case "confirmUndelegate":
		var args ConfirmUndelegateArgs
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

		return nil, self.redelegate(ctx, args)

	case "claimRewards":
		var args ClaimRewardsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimRewards input args: ", err)
			return nil, err
		}

		return nil, self.claimRewards(ctx, args)

	case "claimCommissionRewards":
		var args ClaimCommissionRewardsArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse claimCommissionRewards input args: ", err)
			return nil, err
		}

		return nil, self.claimCommissionRewards(ctx, args)

	case "registerValidator":
		var args RegisterValidatorArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse registerValidator input args: ", err)
			return nil, err
		}

		return nil, self.registerValidator(ctx, args)

	case "setValidatorInfo":
		var args SetValidatorInfoArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse setValidatorInfo input args: ", err)
			return nil, err
		}

		return nil, self.setValidatorInfo(ctx, args)

	case "isValidatorEligible":
		var args IsValidatorEligibleArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse isValidatorEligible input args: ", err)
			return nil, err
		}
		result:= self.DelayedStorage.IsValidatorEligible(&args.Validator)
		return method.Outputs.Pack(result)

	case "getTotalEligibleValidatorsCount":
		var args GetTotalEligibleValidatorsCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getTotalEligibleValidatorsCount input args: ", err)
			return nil, err
		}

		result := self.DelayedStorage.GetTotalEligibleValidatorsCount()
		return method.Outputs.Pack(result)

	case "getTotalEligibleVotesCount":
		var args GetTotalEligibleVotesCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getTotalEligibleVotesCount input args: ", err)
			return nil, err
		}

		result := self.DelayedStorage.GetTotalEligibleVotesCount()
		return method.Outputs.Pack(result)

	case "getValidatorEligibleVotesCount":
		var args GetValidatorEligibleVotesCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}

		result:= self.DelayedStorage.GetValidatorEligibleVotesCount(&args.Validator)
		return method.Outputs.Pack(result)
	}

	return nil, nil
}

// Delegates <amount> of tokens to specified validator
func (self *Contract) delegate(ctx vm.CallFrame, args DelegateArgs) error {
	// // TODO: is the storga echange undo if the tx fails or not ?
	// if !self.Storage.SubBalance(&delegator_addr, amount) {
	// 	return ErrInsufficientBalance
	// }

	// // Checks if validator exists
	// var validator *ValidatorInfo = nil
	// validator, validator_exists := self.ValidatorsInfo[validator_addr]

	// // No such validator in memory - try to get him from storage
	// if validator_exists == false {
	// 	k := stor_k_1(field_validators_info, validator_addr[:])
	// 	self.Storage.Get(k, func(bytes []byte) {
	// 		validator = make(ValidatorInfo)
	// 		rlp.MustDecodeBytes(bytes, validator)
	// 	})

	// 	if validator == nil {
	// 		return ErrNonExistentValidator
	// 	}
	// }

	// // Checks max validator's stake condition
	// if validator.TotalStake+amount > MAX_VALIDATOR_STAKE {
	// 	return ErrValidatorsMaxStakeExceeded
	// }

	// // Checks if validator has such delegator already
	// var delegator *DelegatorInfo = nil
	// delegator, delegator_exists := validator.Delegators[delegator_addr]
	// // No such delegator exists
	// if delegator_exists == false {
	// 	if amount.Cmp(MIN_DELEGATOR_STAKE) < 0 {
	// 		return ErrInsufficientDelegation
	// 	}

	// 	delegator = make(DelegatorInfo)
	// 	delegator.Stake = amount

	// 	// TODO: add validator as delegator's validator and put it into the storage
	// } else {
	// 	if delegator.Stake.Add(amount).Cmp(MIN_DELEGATOR_STAKE) < 0 {
	// 		return ErrInsufficientDelegation
	// 	}

	// 	delegator.Stake = delegator.Stake.Add(amount)
	// }

	// validator.TotalStake = validator.TotalStake.Add(amount)
	// validator.Delegators[delegator_addr] = delegator

	// self.ValidatorsInfo[validator_addr] = validator
	// self.Storage.Put(
	// 	stor_k_1(field_validators_info, validator_addr[:]),
	// 	rlp.MustEncodeToBytes(validator))

	return nil
}

func (self *Contract) undelegate(ctx vm.CallFrame, args UndelegateArgs) error {
	return nil
}

func (self *Contract) confirmUndelegate(ctx vm.CallFrame, args ConfirmUndelegateArgs) error {
	return nil
}

func (self *Contract) redelegate(ctx vm.CallFrame, args RedelegateArgs) error {
	return nil
}

func (self *Contract) claimRewards(ctx vm.CallFrame, args ClaimRewardsArgs) error {
	return nil
}

func (self *Contract) claimCommissionRewards(ctx vm.CallFrame, args ClaimCommissionRewardsArgs) error {
	return nil
}

func (self *Contract) registerValidator(ctx vm.CallFrame, args RegisterValidatorArgs) error {
	return nil
}

func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
	return nil
}

func (self *Contract) setCommission(ctx vm.CallFrame, args SetCommissionArgs) error {
	return nil
}
