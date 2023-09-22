package chain_config

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
)

type Redelegation struct {
	Validator common.Address
	Delegator common.Address
	Amount    *big.Int
}

type MagnoliaHfConfig struct {
	BlockNum uint64
	JailTime uint64 // [number of blocks]
}

type HardforksConfig struct {
	FixRedelegateBlockNum        uint64
	Redelegations                []Redelegation
	RewardsDistributionFrequency map[uint64]uint32
	MagnoliaHf                   MagnoliaHfConfig
}

func (self *HardforksConfig) IsMagnoliaHardfork(block uint64) bool {
	return block >= self.MagnoliaHf.BlockNum
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

type ChainConfig struct {
	EVMChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            DPOSConfig
	Hardforks       HardforksConfig
}

func (self *ChainConfig) RewardsEnabled() bool {
	return self.DPOS.YieldPercentage > 0
}
