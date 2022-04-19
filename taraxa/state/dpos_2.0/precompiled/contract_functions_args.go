package dpos_2

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

// Note: arguments names inside structs must match args names from solidity interface

type DelegateArgs struct {
	Validator common.Address
}

type UndelegateArgs struct {
	Validator common.Address
	Amount    *big.Int
}

type ConfirmUndelegateArgs struct {
	Validator common.Address
}

type RedelegateArgs struct {
	Validator_from common.Address
	Validator_to   common.Address
	Amount         *big.Int
}

type ClaimRewardsArgs struct {
	Validator common.Address
	Amount    *big.Int
}

type ClaimCommissionRewardsArgs struct {
	Amount *big.Int
}

type RegisterValidatorArgs struct {
	Commission   *big.Int
	Descriptions string
	Endpoint     string
}

type SetValidatorInfoArgs struct {
	Descriptions string
	Endpoint     string
}

type SetCommissionArgs struct {
	Commission *big.Int
}

type IsValidatorEligibleArgs struct {
	Block_num *big.Int
	Validator common.Address
}

type GetTotalEligibleValidatorsCountArgs struct {
	Block_num *big.Int
}

type GetTotalEligibleVotesCountArgs struct {
	Block_num *big.Int
}

type GetValidatorEligibleVotesCountArgs struct {
	Block_num *big.Int
	Validator common.Address
}

type GetValidatorArgs struct {
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
