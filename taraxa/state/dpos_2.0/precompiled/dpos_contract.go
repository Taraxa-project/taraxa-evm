package dpos_2

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
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
type Validator struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission *big.Int

	// Rewards accumulated from delegators rewards based on commission
	CommissionRewards *big.Int

	// Short description
	// TODO: optional - maybe we dont want this ?
	//Description string

	// Validator's website url
	// TODO: optional - maybe we dont want this ?
	//Endpoint string
}

type ValidatorDelegators struct {
	// List of validator's delegators
	Delegators map[common.Address]Delegator
}

type Delegator struct {
	// Num of delegated tokens == delegator's stake
	Stake *big.Int

	// UnlockedStake == unlocked(undelegated) tokens that can be withdrawn now
	// TODO: in case we will send unlocked tokens to the delegator's balance automatically, we dont need this field
	UnlockedStake *big.Int

	// Accumulated rewards
	Rewards *big.Int

	// Undelegate request
	// TODO: rethink implementation of undelegations
	//UndelegateRequests []*UndelegateRequest
}

func (delegator Delegator) Serialize() (result []byte) {
	// TODO: serialize delegator into the byte array

	return result
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

type Contract struct {
	Storage StorageWrapper
	Abi     abi.ABI

	// TODO: mappings that are going to be in storage
	// 			 1. field_validators + validator_address -> Serialized(Validator) // non-iterable map thath holds basic validator info // can be loaded during init based on 3.
	//			 2. field_validators_delegators + validator_address -> Serialized(ValidatorDelegators) // non-iterable map that holds delegator(who delegated to validator_address) info // can be loaded during init based on 3.
	// 			 3. IterableMap(field_validators_iter_map) // iterbale map of address of all validators, real data have to be fetched from 1. and 2. maps // can be loaded during init
	//			 4. IterableMap(field_delegators_validators_iter_map + delegator_address) // iterbale map of address of all validators that delegator delegated to, real data have to be fetched from 1. and 2. maps
	//			 4'. delegators_validators map[common.Address]*IterableMap(field_delegators_validators_iter_map + delegator_address) // cannot be loaded during init - has to be created ad-hod if it does not exist in delegators_validators

	// Storage - related structures
	ValidatorsIterMap    IterableMap
	DelegatorsValidators map[common.Address]IterableMap

	// Validadors basic info, e.g. total stake, commission, etc...
	// ValidatorsInfo map[common.Address]*ValidatorInfo

	// // Delegators list of their validators -> key = delegator address
	// DelegatorsValidators map[common.Address]*DelegatorValidators
}

var (
	field_validators                     = []byte{0}
	field_validators_delegators          = []byte{1}
	field_validators_iter_map            = []byte{2}
	field_delegators_validators_iter_map = []byte{3}

	//...
)

func (self *Contract) Init(storage Storage, last_commited_block_num types.BlockNum) *Contract {
	self.Storage.Init(storage)

	dpos_abi, err := ioutil.ReadFile("DposInterface.abi")
	if err != nil {
		panic("Unable to load dpos contract interface abi: " + err.Error())
	}
	self.Abi, _ = abi.JSON(strings.NewReader(string(dpos_abi)))

	self.ValidatorsIterMap.Init(self.Storage, field_validators_iter_map)

	// TODO: read delayedRequest from storage for blocks <last_commited_block_num+1, last_commited_block_num + 1 + delay>

	return self
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

		result, err := self.isValidatorEligible(ctx, args)
		if err != nil {
			fmt.Println("isValidatorEligible processing error: ", err)
			return nil, err
		}

		return method.Outputs.Pack(result)

	case "getTotalEligibleValidatorsCount":
		var args GetTotalEligibleValidatorsCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getTotalEligibleValidatorsCount input args: ", err)
			return nil, err
		}

		result, err := self.getTotalEligibleValidatorsCount(ctx, args)
		if err != nil {
			fmt.Println("getTotalEligibleValidatorsCount processing error: ", err)
			return nil, err
		}

		return method.Outputs.Pack(result)

	case "getTotalEligibleVotesCount":
		var args GetTotalEligibleVotesCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getTotalEligibleVotesCount input args: ", err)
			return nil, err
		}

		result, err := self.getTotalEligibleVotesCount(ctx, args)
		if err != nil {
			fmt.Println("getTotalEligibleVotesCount processing error: ", err)
			return nil, err
		}

		return method.Outputs.Pack(result)

	case "getValidatorEligibleVotesCount":
		var args GetValidatorEligibleVotesCountArgs
		if err = method.Inputs.Unpack(&args, input); err != nil {
			fmt.Println("Unable to parse getValidatorEligibleVotesCount input args: ", err)
			return nil, err
		}

		result, err := self.getValidatorEligibleVotesCount(ctx, args)
		if err != nil {
			fmt.Println("getValidatorEligibleVotesCount processing error: ", err)
			return nil, err
		}

		return method.Outputs.Pack(result)
	}

	return nil, nil
}

func (self *Contract) createValidator(validator_addr common.Address, stake *big.Int, commission *big.Int) error {
	validators_k := stor_k_1(field_validators, validator_addr[:])
	validator := Validator{stake, commission, big.NewInt(0)}
	self.Storage.Put(validators_k, rlp.MustEncodeToBytes(validator))

	self.ValidatorsIterMap.CreateAccount(validator_addr)

	// Validator must also be a delegator to himself(has to have some self stake)
	self.createDelegator(validator_addr, validator_addr, stake)

	return nil
}

func (self *Contract) createDelegator(validator_addr common.Address, delegator_addr common.Address, stake *big.Int) error {
	validator_delegators_k := stor_k_1(field_validators_delegators, validator_addr[:])
	var validator_delegators ValidatorDelegators

	// Loads validator delegators from storage
	self.Storage.Get(validator_delegators_k, func(bytes []byte) {
		rlp.MustDecodeBytes(bytes, &validator_delegators)
	})

	validator_delegators.Delegators[delegator_addr] = Delegator{stake, big.NewInt(0), big.NewInt(0)}

	// Save adjusted validator delegators into storage
	self.Storage.Put(validator_delegators_k, rlp.MustEncodeToBytes(validator_delegators))

	return nil
}

func (self *Contract) addDelegatorsStake(validator_addr common.Address, delegator_addr common.Address, stake *big.Int) error {
	validator_delegators_k := stor_k_1(field_validators_delegators, validator_addr[:])
	var validator_delegators ValidatorDelegators

	// Loads validator delegators from storage
	self.Storage.Get(validator_delegators_k, func(bytes []byte) {
		rlp.MustDecodeBytes(bytes, &validator_delegators)
	})

	if delegator, found := validator_delegators.Delegators[delegator_addr]; found {
		delegator.Stake = bigutil.Add(delegator.Stake, stake)
		validator_delegators.Delegators[delegator_addr] = delegator

		// Save adjusted validator delegators into storage
		self.Storage.Put(validator_delegators_k, rlp.MustEncodeToBytes(validator_delegators))
		return nil
	}

	// This should never happen
	panic("Delegator not found")
}

// Delegates <amount> of tokens to specified validator
func (self *Contract) delegate(ctx vm.CallFrame, args DelegateArgs) error {
	// TODO: some checks for insufficient balance, etc...

	// Checks if validator exists
	if self.ValidatorsIterMap.AccountExists(args.Validator) == false {
		return util.ErrorString("Non-existent validator")
	}

	// Loads iterable map of validator for specified delegator
	var delegator_validators_iter_map IterableMap
	if delegator_validators_iter_map, found := self.DelegatorsValidators[*ctx.Account.Address()]; !found {
		delegator_validators_iter_map = IterableMap{}
		delegator_validators_iter_map.Init(self.Storage, append(field_delegators_validators_iter_map, ctx.Account.Address().Bytes()...))

		self.DelegatorsValidators[*ctx.Account.Address()] = delegator_validators_iter_map
	}

	// Checks if delegator exists
	if delegator_validators_iter_map.AccountExists(args.Validator) == false {
		self.createDelegator(args.Validator, *ctx.Account.Address(), ctx.Value)
	} else {
		self.addDelegatorsStake(args.Validator, *ctx.Account.Address(), ctx.Value)
	}

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
	if self.ValidatorsIterMap.AccountExists(*ctx.Account.Address()) == true {
		return util.ErrorString("Validator already exists")
	}

	// TODO: check all conditions

	self.createValidator(*ctx.Account.Address(), ctx.Value, args.Commission)

	return nil
}

func (self *Contract) setValidatorInfo(ctx vm.CallFrame, args SetValidatorInfoArgs) error {
	return nil
}

func (self *Contract) setCommission(ctx vm.CallFrame, args SetCommissionArgs) error {
	return nil
}

func (self *Contract) isValidatorEligible(ctx vm.CallFrame, args IsValidatorEligibleArgs) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (self *Contract) getTotalEligibleValidatorsCount(ctx vm.CallFrame, args GetTotalEligibleValidatorsCountArgs) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (self *Contract) getTotalEligibleVotesCount(ctx vm.CallFrame, args GetTotalEligibleVotesCountArgs) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (self *Contract) getValidatorEligibleVotesCount(ctx vm.CallFrame, args GetValidatorEligibleVotesCountArgs) (*big.Int, error) {
	return big.NewInt(0), nil
}
