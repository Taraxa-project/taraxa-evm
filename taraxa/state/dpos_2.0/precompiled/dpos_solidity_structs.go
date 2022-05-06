// This file was was created manually with some parts generated automatically and copy pasted
// For automatic generation & copy paste:
//		 1. cd ../solidity
//		 2. abigen --abi=abi/DposInterface.abi --pkg=taraxaDposClient --out=dpos_contract_interface.go
//		 3. copy selected structs into this file
// 		 4. rm dpos_contract_interface.go

package dpos_2

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

// Automatically generated & Copy pasted structs

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

// DposInterfaceValidatorBasicInfo is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorBasicInfo struct {
	TotalStake       *big.Int
	Commission       uint16
	CommissionReward *big.Int
	Description      string
	Endpoint         string
}

// DposInterfaceValidatorData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorData struct {
	Account common.Address
	Info    DposInterfaceValidatorBasicInfo
}

// Manually created structs
type GetValidatorsRet struct {
	Validators []DposInterfaceValidatorData
	End        bool
}

type GetDelegatorDelegationRet struct {
	Delegations []DposInterfaceDelegationData
	End         bool
}

// Note: arguments names inside structs must match args names from solidity interface
type UndelegateArgs struct {
	Validator common.Address
	Amount    *big.Int
}

type RedelegateArgs struct {
	Validator_from common.Address
	Validator_to   common.Address
	Amount         *big.Int
}

type RegisterValidatorArgs struct {
	Commission  uint16
	Description string
	Endpoint    string
}

type SetValidatorInfoArgs struct {
	Description string
	Endpoint    string
}

type SetCommissionArgs struct {
	Commission uint16
}

type ValidatorAddress struct {
	Validator common.Address
}

type GetValidatorsArgs struct {
	Batch uint32
}

type GetDelegatorDelegationsArgs struct {
	Delegator common.Address
	Batch     uint32
}
