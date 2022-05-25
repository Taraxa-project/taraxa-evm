// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package taraxaDposClient

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// DposInterfaceDelegationData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceDelegationData struct {
	Account    common.Address
	Delegation DposInterfaceDelegatorInfo
}

// DposInterfaceDelegatorInfo is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceDelegatorInfo struct {
	Stake   *big.Int
	Rewards *big.Int
}

// DposInterfaceUndelegationData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceUndelegationData struct {
	Stake     *big.Int
	Block     uint64
	Validator common.Address
}

// DposInterfaceValidatorBasicInfo is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorBasicInfo struct {
	TotalStake       *big.Int
	CommissionReward *big.Int
	Commission       uint16
	Description      string
	Endpoint         string
}

// DposInterfaceValidatorData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorData struct {
	Account common.Address
	Info    DposInterfaceValidatorBasicInfo
}

// TaraxaDposClientMetaData contains all meta data concerning the TaraxaDposClient contract.
var TaraxaDposClientMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"cancelUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimCommissionRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"confirmUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getDelegatorDelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"internalType\":\"structDposInterface.DelegatorInfo\",\"name\":\"delegation\",\"type\":\"tuple\"}],\"internalType\":\"structDposInterface.DelegationData[]\",\"name\":\"delegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalEligibleValidatorsCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"internalType\":\"structDposInterface.UndelegationData[]\",\"name\":\"undelegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidator\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"structDposInterface.ValidatorBasicInfo\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidatorEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidators\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"structDposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"structDposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isValidatorEligible\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"validator_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"reDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"registerValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"setCommission\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"setValidatorInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// TaraxaDposClientABI is the input ABI used to generate the binding from.
// Deprecated: Use TaraxaDposClientMetaData.ABI instead.
var TaraxaDposClientABI = TaraxaDposClientMetaData.ABI

// TaraxaDposClient is an auto generated Go binding around an Ethereum contract.
type TaraxaDposClient struct {
	TaraxaDposClientCaller     // Read-only binding to the contract
	TaraxaDposClientTransactor // Write-only binding to the contract
	TaraxaDposClientFilterer   // Log filterer for contract events
}

// TaraxaDposClientCaller is an auto generated read-only Go binding around an Ethereum contract.
type TaraxaDposClientCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TaraxaDposClientTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TaraxaDposClientTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TaraxaDposClientFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TaraxaDposClientFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TaraxaDposClientSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TaraxaDposClientSession struct {
	Contract     *TaraxaDposClient // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TaraxaDposClientCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TaraxaDposClientCallerSession struct {
	Contract *TaraxaDposClientCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// TaraxaDposClientTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TaraxaDposClientTransactorSession struct {
	Contract     *TaraxaDposClientTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// TaraxaDposClientRaw is an auto generated low-level Go binding around an Ethereum contract.
type TaraxaDposClientRaw struct {
	Contract *TaraxaDposClient // Generic contract binding to access the raw methods on
}

// TaraxaDposClientCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TaraxaDposClientCallerRaw struct {
	Contract *TaraxaDposClientCaller // Generic read-only contract binding to access the raw methods on
}

// TaraxaDposClientTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TaraxaDposClientTransactorRaw struct {
	Contract *TaraxaDposClientTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTaraxaDposClient creates a new instance of TaraxaDposClient, bound to a specific deployed contract.
func NewTaraxaDposClient(address common.Address, backend bind.ContractBackend) (*TaraxaDposClient, error) {
	contract, err := bindTaraxaDposClient(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TaraxaDposClient{TaraxaDposClientCaller: TaraxaDposClientCaller{contract: contract}, TaraxaDposClientTransactor: TaraxaDposClientTransactor{contract: contract}, TaraxaDposClientFilterer: TaraxaDposClientFilterer{contract: contract}}, nil
}

// NewTaraxaDposClientCaller creates a new read-only instance of TaraxaDposClient, bound to a specific deployed contract.
func NewTaraxaDposClientCaller(address common.Address, caller bind.ContractCaller) (*TaraxaDposClientCaller, error) {
	contract, err := bindTaraxaDposClient(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TaraxaDposClientCaller{contract: contract}, nil
}

// NewTaraxaDposClientTransactor creates a new write-only instance of TaraxaDposClient, bound to a specific deployed contract.
func NewTaraxaDposClientTransactor(address common.Address, transactor bind.ContractTransactor) (*TaraxaDposClientTransactor, error) {
	contract, err := bindTaraxaDposClient(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TaraxaDposClientTransactor{contract: contract}, nil
}

// NewTaraxaDposClientFilterer creates a new log filterer instance of TaraxaDposClient, bound to a specific deployed contract.
func NewTaraxaDposClientFilterer(address common.Address, filterer bind.ContractFilterer) (*TaraxaDposClientFilterer, error) {
	contract, err := bindTaraxaDposClient(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TaraxaDposClientFilterer{contract: contract}, nil
}

// bindTaraxaDposClient binds a generic wrapper to an already deployed contract.
func bindTaraxaDposClient(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TaraxaDposClientABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TaraxaDposClient *TaraxaDposClientRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TaraxaDposClient.Contract.TaraxaDposClientCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TaraxaDposClient *TaraxaDposClientRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.TaraxaDposClientTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TaraxaDposClient *TaraxaDposClientRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.TaraxaDposClientTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TaraxaDposClient *TaraxaDposClientCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TaraxaDposClient.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TaraxaDposClient *TaraxaDposClientTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TaraxaDposClient *TaraxaDposClientTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.contract.Transact(opts, method, params...)
}

// GetDelegatorDelegations is a free data retrieval call binding the contract method 0xb1621eed.
//
// Solidity: function getDelegatorDelegations(address delegator, uint32 batch) view returns((address,(uint256,uint256))[] delegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetDelegatorDelegations(opts *bind.CallOpts, delegator common.Address, batch uint32) (struct {
	Delegations []DposInterfaceDelegationData
	End         bool
}, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getDelegatorDelegations", delegator, batch)

	outstruct := new(struct {
		Delegations []DposInterfaceDelegationData
		End         bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Delegations = *abi.ConvertType(out[0], new([]DposInterfaceDelegationData)).(*[]DposInterfaceDelegationData)
	outstruct.End = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// GetDelegatorDelegations is a free data retrieval call binding the contract method 0xb1621eed.
//
// Solidity: function getDelegatorDelegations(address delegator, uint32 batch) view returns((address,(uint256,uint256))[] delegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientSession) GetDelegatorDelegations(delegator common.Address, batch uint32) (struct {
	Delegations []DposInterfaceDelegationData
	End         bool
}, error) {
	return _TaraxaDposClient.Contract.GetDelegatorDelegations(&_TaraxaDposClient.CallOpts, delegator, batch)
}

// GetDelegatorDelegations is a free data retrieval call binding the contract method 0xb1621eed.
//
// Solidity: function getDelegatorDelegations(address delegator, uint32 batch) view returns((address,(uint256,uint256))[] delegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetDelegatorDelegations(delegator common.Address, batch uint32) (struct {
	Delegations []DposInterfaceDelegationData
	End         bool
}, error) {
	return _TaraxaDposClient.Contract.GetDelegatorDelegations(&_TaraxaDposClient.CallOpts, delegator, batch)
}

// GetTotalEligibleValidatorsCount is a free data retrieval call binding the contract method 0x8de1fbbe.
//
// Solidity: function getTotalEligibleValidatorsCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetTotalEligibleValidatorsCount(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getTotalEligibleValidatorsCount")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetTotalEligibleValidatorsCount is a free data retrieval call binding the contract method 0x8de1fbbe.
//
// Solidity: function getTotalEligibleValidatorsCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientSession) GetTotalEligibleValidatorsCount() (uint64, error) {
	return _TaraxaDposClient.Contract.GetTotalEligibleValidatorsCount(&_TaraxaDposClient.CallOpts)
}

// GetTotalEligibleValidatorsCount is a free data retrieval call binding the contract method 0x8de1fbbe.
//
// Solidity: function getTotalEligibleValidatorsCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetTotalEligibleValidatorsCount() (uint64, error) {
	return _TaraxaDposClient.Contract.GetTotalEligibleValidatorsCount(&_TaraxaDposClient.CallOpts)
}

// GetTotalEligibleVotesCount is a free data retrieval call binding the contract method 0xde8e4b50.
//
// Solidity: function getTotalEligibleVotesCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetTotalEligibleVotesCount(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getTotalEligibleVotesCount")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetTotalEligibleVotesCount is a free data retrieval call binding the contract method 0xde8e4b50.
//
// Solidity: function getTotalEligibleVotesCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientSession) GetTotalEligibleVotesCount() (uint64, error) {
	return _TaraxaDposClient.Contract.GetTotalEligibleVotesCount(&_TaraxaDposClient.CallOpts)
}

// GetTotalEligibleVotesCount is a free data retrieval call binding the contract method 0xde8e4b50.
//
// Solidity: function getTotalEligibleVotesCount() view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetTotalEligibleVotesCount() (uint64, error) {
	return _TaraxaDposClient.Contract.GetTotalEligibleVotesCount(&_TaraxaDposClient.CallOpts)
}

// GetUndelegations is a free data retrieval call binding the contract method 0x4edd9943.
//
// Solidity: function getUndelegations(address delegator, uint32 batch) view returns((uint256,uint64,address)[] undelegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetUndelegations(opts *bind.CallOpts, delegator common.Address, batch uint32) (struct {
	Undelegations []DposInterfaceUndelegationData
	End           bool
}, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getUndelegations", delegator, batch)

	outstruct := new(struct {
		Undelegations []DposInterfaceUndelegationData
		End           bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Undelegations = *abi.ConvertType(out[0], new([]DposInterfaceUndelegationData)).(*[]DposInterfaceUndelegationData)
	outstruct.End = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// GetUndelegations is a free data retrieval call binding the contract method 0x4edd9943.
//
// Solidity: function getUndelegations(address delegator, uint32 batch) view returns((uint256,uint64,address)[] undelegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientSession) GetUndelegations(delegator common.Address, batch uint32) (struct {
	Undelegations []DposInterfaceUndelegationData
	End           bool
}, error) {
	return _TaraxaDposClient.Contract.GetUndelegations(&_TaraxaDposClient.CallOpts, delegator, batch)
}

// GetUndelegations is a free data retrieval call binding the contract method 0x4edd9943.
//
// Solidity: function getUndelegations(address delegator, uint32 batch) view returns((uint256,uint64,address)[] undelegations, bool end)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetUndelegations(delegator common.Address, batch uint32) (struct {
	Undelegations []DposInterfaceUndelegationData
	End           bool
}, error) {
	return _TaraxaDposClient.Contract.GetUndelegations(&_TaraxaDposClient.CallOpts, delegator, batch)
}

// GetValidator is a free data retrieval call binding the contract method 0x1904bb2e.
//
// Solidity: function getValidator(address validator) view returns((uint256,uint256,uint16,string,string))
func (_TaraxaDposClient *TaraxaDposClientCaller) GetValidator(opts *bind.CallOpts, validator common.Address) (DposInterfaceValidatorBasicInfo, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getValidator", validator)

	if err != nil {
		return *new(DposInterfaceValidatorBasicInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(DposInterfaceValidatorBasicInfo)).(*DposInterfaceValidatorBasicInfo)

	return out0, err

}

// GetValidator is a free data retrieval call binding the contract method 0x1904bb2e.
//
// Solidity: function getValidator(address validator) view returns((uint256,uint256,uint16,string,string))
func (_TaraxaDposClient *TaraxaDposClientSession) GetValidator(validator common.Address) (DposInterfaceValidatorBasicInfo, error) {
	return _TaraxaDposClient.Contract.GetValidator(&_TaraxaDposClient.CallOpts, validator)
}

// GetValidator is a free data retrieval call binding the contract method 0x1904bb2e.
//
// Solidity: function getValidator(address validator) view returns((uint256,uint256,uint16,string,string))
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetValidator(validator common.Address) (DposInterfaceValidatorBasicInfo, error) {
	return _TaraxaDposClient.Contract.GetValidator(&_TaraxaDposClient.CallOpts, validator)
}

// GetValidatorEligibleVotesCount is a free data retrieval call binding the contract method 0x618e3862.
//
// Solidity: function getValidatorEligibleVotesCount(address validator) view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetValidatorEligibleVotesCount(opts *bind.CallOpts, validator common.Address) (uint64, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getValidatorEligibleVotesCount", validator)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetValidatorEligibleVotesCount is a free data retrieval call binding the contract method 0x618e3862.
//
// Solidity: function getValidatorEligibleVotesCount(address validator) view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientSession) GetValidatorEligibleVotesCount(validator common.Address) (uint64, error) {
	return _TaraxaDposClient.Contract.GetValidatorEligibleVotesCount(&_TaraxaDposClient.CallOpts, validator)
}

// GetValidatorEligibleVotesCount is a free data retrieval call binding the contract method 0x618e3862.
//
// Solidity: function getValidatorEligibleVotesCount(address validator) view returns(uint64)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetValidatorEligibleVotesCount(validator common.Address) (uint64, error) {
	return _TaraxaDposClient.Contract.GetValidatorEligibleVotesCount(&_TaraxaDposClient.CallOpts, validator)
}

// GetValidators is a free data retrieval call binding the contract method 0x19d8024f.
//
// Solidity: function getValidators(uint32 batch) view returns((address,(uint256,uint256,uint16,string,string))[] validators, bool end)
func (_TaraxaDposClient *TaraxaDposClientCaller) GetValidators(opts *bind.CallOpts, batch uint32) (struct {
	Validators []DposInterfaceValidatorData
	End        bool
}, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "getValidators", batch)

	outstruct := new(struct {
		Validators []DposInterfaceValidatorData
		End        bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Validators = *abi.ConvertType(out[0], new([]DposInterfaceValidatorData)).(*[]DposInterfaceValidatorData)
	outstruct.End = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// GetValidators is a free data retrieval call binding the contract method 0x19d8024f.
//
// Solidity: function getValidators(uint32 batch) view returns((address,(uint256,uint256,uint16,string,string))[] validators, bool end)
func (_TaraxaDposClient *TaraxaDposClientSession) GetValidators(batch uint32) (struct {
	Validators []DposInterfaceValidatorData
	End        bool
}, error) {
	return _TaraxaDposClient.Contract.GetValidators(&_TaraxaDposClient.CallOpts, batch)
}

// GetValidators is a free data retrieval call binding the contract method 0x19d8024f.
//
// Solidity: function getValidators(uint32 batch) view returns((address,(uint256,uint256,uint16,string,string))[] validators, bool end)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) GetValidators(batch uint32) (struct {
	Validators []DposInterfaceValidatorData
	End        bool
}, error) {
	return _TaraxaDposClient.Contract.GetValidators(&_TaraxaDposClient.CallOpts, batch)
}

// IsValidatorEligible is a free data retrieval call binding the contract method 0xf3094e90.
//
// Solidity: function isValidatorEligible(address validator) view returns(bool)
func (_TaraxaDposClient *TaraxaDposClientCaller) IsValidatorEligible(opts *bind.CallOpts, validator common.Address) (bool, error) {
	var out []interface{}
	err := _TaraxaDposClient.contract.Call(opts, &out, "isValidatorEligible", validator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidatorEligible is a free data retrieval call binding the contract method 0xf3094e90.
//
// Solidity: function isValidatorEligible(address validator) view returns(bool)
func (_TaraxaDposClient *TaraxaDposClientSession) IsValidatorEligible(validator common.Address) (bool, error) {
	return _TaraxaDposClient.Contract.IsValidatorEligible(&_TaraxaDposClient.CallOpts, validator)
}

// IsValidatorEligible is a free data retrieval call binding the contract method 0xf3094e90.
//
// Solidity: function isValidatorEligible(address validator) view returns(bool)
func (_TaraxaDposClient *TaraxaDposClientCallerSession) IsValidatorEligible(validator common.Address) (bool, error) {
	return _TaraxaDposClient.Contract.IsValidatorEligible(&_TaraxaDposClient.CallOpts, validator)
}

// CancelUndelegate is a paid mutator transaction binding the contract method 0x399ff554.
//
// Solidity: function cancelUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) CancelUndelegate(opts *bind.TransactOpts, validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "cancelUndelegate", validator)
}

// CancelUndelegate is a paid mutator transaction binding the contract method 0x399ff554.
//
// Solidity: function cancelUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) CancelUndelegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.CancelUndelegate(&_TaraxaDposClient.TransactOpts, validator)
}

// CancelUndelegate is a paid mutator transaction binding the contract method 0x399ff554.
//
// Solidity: function cancelUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) CancelUndelegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.CancelUndelegate(&_TaraxaDposClient.TransactOpts, validator)
}

// ClaimCommissionRewards is a paid mutator transaction binding the contract method 0xe51942dc.
//
// Solidity: function claimCommissionRewards() returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) ClaimCommissionRewards(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "claimCommissionRewards")
}

// ClaimCommissionRewards is a paid mutator transaction binding the contract method 0xe51942dc.
//
// Solidity: function claimCommissionRewards() returns()
func (_TaraxaDposClient *TaraxaDposClientSession) ClaimCommissionRewards() (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ClaimCommissionRewards(&_TaraxaDposClient.TransactOpts)
}

// ClaimCommissionRewards is a paid mutator transaction binding the contract method 0xe51942dc.
//
// Solidity: function claimCommissionRewards() returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) ClaimCommissionRewards() (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ClaimCommissionRewards(&_TaraxaDposClient.TransactOpts)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xef5cfb8c.
//
// Solidity: function claimRewards(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) ClaimRewards(opts *bind.TransactOpts, validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "claimRewards", validator)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xef5cfb8c.
//
// Solidity: function claimRewards(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) ClaimRewards(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ClaimRewards(&_TaraxaDposClient.TransactOpts, validator)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xef5cfb8c.
//
// Solidity: function claimRewards(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) ClaimRewards(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ClaimRewards(&_TaraxaDposClient.TransactOpts, validator)
}

// ConfirmUndelegate is a paid mutator transaction binding the contract method 0x45a02561.
//
// Solidity: function confirmUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) ConfirmUndelegate(opts *bind.TransactOpts, validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "confirmUndelegate", validator)
}

// ConfirmUndelegate is a paid mutator transaction binding the contract method 0x45a02561.
//
// Solidity: function confirmUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) ConfirmUndelegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ConfirmUndelegate(&_TaraxaDposClient.TransactOpts, validator)
}

// ConfirmUndelegate is a paid mutator transaction binding the contract method 0x45a02561.
//
// Solidity: function confirmUndelegate(address validator) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) ConfirmUndelegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ConfirmUndelegate(&_TaraxaDposClient.TransactOpts, validator)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address validator) payable returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) Delegate(opts *bind.TransactOpts, validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "delegate", validator)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address validator) payable returns()
func (_TaraxaDposClient *TaraxaDposClientSession) Delegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.Delegate(&_TaraxaDposClient.TransactOpts, validator)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address validator) payable returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) Delegate(validator common.Address) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.Delegate(&_TaraxaDposClient.TransactOpts, validator)
}

// ReDelegate is a paid mutator transaction binding the contract method 0x703812cc.
//
// Solidity: function reDelegate(address validator_from, address validator_to, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) ReDelegate(opts *bind.TransactOpts, validator_from common.Address, validator_to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "reDelegate", validator_from, validator_to, amount)
}

// ReDelegate is a paid mutator transaction binding the contract method 0x703812cc.
//
// Solidity: function reDelegate(address validator_from, address validator_to, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) ReDelegate(validator_from common.Address, validator_to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ReDelegate(&_TaraxaDposClient.TransactOpts, validator_from, validator_to, amount)
}

// ReDelegate is a paid mutator transaction binding the contract method 0x703812cc.
//
// Solidity: function reDelegate(address validator_from, address validator_to, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) ReDelegate(validator_from common.Address, validator_to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.ReDelegate(&_TaraxaDposClient.TransactOpts, validator_from, validator_to, amount)
}

// RegisterValidator is a paid mutator transaction binding the contract method 0xb57d576a.
//
// Solidity: function registerValidator(uint16 commission, string description, string endpoint) payable returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) RegisterValidator(opts *bind.TransactOpts, commission uint16, description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "registerValidator", commission, description, endpoint)
}

// RegisterValidator is a paid mutator transaction binding the contract method 0xb57d576a.
//
// Solidity: function registerValidator(uint16 commission, string description, string endpoint) payable returns()
func (_TaraxaDposClient *TaraxaDposClientSession) RegisterValidator(commission uint16, description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.RegisterValidator(&_TaraxaDposClient.TransactOpts, commission, description, endpoint)
}

// RegisterValidator is a paid mutator transaction binding the contract method 0xb57d576a.
//
// Solidity: function registerValidator(uint16 commission, string description, string endpoint) payable returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) RegisterValidator(commission uint16, description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.RegisterValidator(&_TaraxaDposClient.TransactOpts, commission, description, endpoint)
}

// SetCommission is a paid mutator transaction binding the contract method 0xc594cc65.
//
// Solidity: function setCommission(uint16 commission) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) SetCommission(opts *bind.TransactOpts, commission uint16) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "setCommission", commission)
}

// SetCommission is a paid mutator transaction binding the contract method 0xc594cc65.
//
// Solidity: function setCommission(uint16 commission) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) SetCommission(commission uint16) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.SetCommission(&_TaraxaDposClient.TransactOpts, commission)
}

// SetCommission is a paid mutator transaction binding the contract method 0xc594cc65.
//
// Solidity: function setCommission(uint16 commission) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) SetCommission(commission uint16) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.SetCommission(&_TaraxaDposClient.TransactOpts, commission)
}

// SetValidatorInfo is a paid mutator transaction binding the contract method 0xf06f67e2.
//
// Solidity: function setValidatorInfo(string description, string endpoint) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) SetValidatorInfo(opts *bind.TransactOpts, description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "setValidatorInfo", description, endpoint)
}

// SetValidatorInfo is a paid mutator transaction binding the contract method 0xf06f67e2.
//
// Solidity: function setValidatorInfo(string description, string endpoint) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) SetValidatorInfo(description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.SetValidatorInfo(&_TaraxaDposClient.TransactOpts, description, endpoint)
}

// SetValidatorInfo is a paid mutator transaction binding the contract method 0xf06f67e2.
//
// Solidity: function setValidatorInfo(string description, string endpoint) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) SetValidatorInfo(description string, endpoint string) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.SetValidatorInfo(&_TaraxaDposClient.TransactOpts, description, endpoint)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4d99dd16.
//
// Solidity: function undelegate(address validator, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactor) Undelegate(opts *bind.TransactOpts, validator common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.contract.Transact(opts, "undelegate", validator, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4d99dd16.
//
// Solidity: function undelegate(address validator, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientSession) Undelegate(validator common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.Undelegate(&_TaraxaDposClient.TransactOpts, validator, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4d99dd16.
//
// Solidity: function undelegate(address validator, uint256 amount) returns()
func (_TaraxaDposClient *TaraxaDposClientTransactorSession) Undelegate(validator common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TaraxaDposClient.Contract.Undelegate(&_TaraxaDposClient.TransactOpts, validator, amount)
}
