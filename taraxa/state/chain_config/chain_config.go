package chain_config

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/solidity"
)

type Redelegation struct {
	Validator common.Address
	Delegator common.Address
	Amount    *big.Int
}

type Hardforks struct {
	FixRedelegateBlockNum uint64
	Redelegations         []Redelegation
}

type ChainConfig struct {
	EVMChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            DPOSConfig
	Hardforks       Hardforks
}

func (self *ChainConfig) RewardsEnabled() bool {
	return self.DPOS.YieldPercentage > 0
}

type DPOSConfig = struct {
	EligibilityBalanceThreshold *big.Int
	VoteEligibilityBalanceStep  *big.Int
	ValidatorMaximumStake       *big.Int
	MinimumDeposit              *big.Int
	MaxBlockAuthorReward        uint16
	DagProposersReward          uint16
	CommissionChangeDelta       uint16
	CommissionChangeFrequency   uint32 // [number of blocks]
	DelegationDelay             uint32 // [number of blocks]
	DelegationLockingPeriod     uint32 // [number of blocks]
	BlocksPerYear               uint32 // [count]
	YieldPercentage             uint16 // [%]
	InitialValidators           []GenesisValidator
}

type DposConfigWithBlock struct {
	DposConfig DPOSConfig
	Blk_n      types.BlockNum
}

type GenesisValidator struct {
	Address     common.Address
	Owner       common.Address
	VrfKey      []byte
	Commission  uint16
	Endpoint    string
	Description string
	Delegations core.BalanceMap
}

func (self *GenesisValidator) GenRegisterValidatorArgs() (vi sol.RegisterValidatorArgs) {
	vi.VrfKey = self.VrfKey
	vi.Commission = self.Commission
	vi.Description = self.Description
	vi.Endpoint = self.Endpoint
	vi.Validator = self.Address
	return
}
