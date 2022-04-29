package dpos_2

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

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
	Commission   uint16
	Descriptions string
	Endpoint     string
}

type SetValidatorInfoArgs struct {
	Descriptions string
	Endpoint     string
}

type SetCommissionArgs struct {
	Commission uint16
}

type ValidatorArgs struct {
	Validator common.Address
}

type GetValidatorsArgs struct {
	Batch *big.Int
}

type GetDelegatorDelegationsArgs struct {
	Delegator common.Address
	Batch     *big.Int
}

type GetValidatorDelegationsArgs struct {
	Validator common.Address
	Batch     *big.Int
}
