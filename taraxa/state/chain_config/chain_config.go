package chain_config

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
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

type AspenHfConfig struct {
	BlockNum  uint64
	MaxSupply *big.Int
}

type HardforksConfig struct {
	FixRedelegateBlockNum        uint64
	Redelegations                []Redelegation
	RewardsDistributionFrequency map[uint64]uint32
	MagnoliaHf                   MagnoliaHfConfig
	AspenHf                      AspenHfConfig
}

func (c *HardforksConfig) IsMagnoliaHardfork(block types.BlockNum) bool {
	return block >= c.MagnoliaHf.BlockNum
}

func (c *HardforksConfig) IsAspenHardfork(block types.BlockNum) bool {
	return block >= c.AspenHf.BlockNum
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

func (c *HardforksConfig) Rules(num types.BlockNum) vm.Rules {
	return vm.Rules{
		IsMagnolia: isForked(c.MagnoliaHf.BlockNum, num),
		IsAspen:    isForked(c.AspenHf.BlockNum, num),
	}
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

func (self *ChainConfig) GenesisBalancesSum() *big.Int {
	sum := big.NewInt(0)
	for _, balance := range self.GenesisBalances {
		sum.Add(sum, balance)
	}

	return sum
}
