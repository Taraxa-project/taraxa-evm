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
	BlockNumPartOne  uint64 // part 1 just starts to save minted tokens (rewards) in db
	BlockNumPartTwo  uint64 // part 2 implements new dynamic yield curve
	MaxSupply        *big.Int
	GeneratedRewards *big.Int // Total number of generated rewards between block 0 and AspenHf BlockNum
}

type FicusHfConfig struct {
	BlockNum              uint64
	PillarBlocksInterval  uint64 // [number of blocks]
	BridgeContractAddress common.Address
}

type CornusHfConfig struct {
	BlockNum                uint64
	DelegationLockingPeriod uint32 // [number of blocks]
	DagGasLimit             uint64
	PbftGasLimit            uint64
}

type SoleiroliaHfConfig struct {
	BlockNum       uint64
	TrxMinGasPrice uint64 // [wei]
	TrxMaxGasLimit uint64
}

// Leaving it here for next HF
// type BambooRedelegation struct {
// 	Validator common.Address
// 	Amount    *big.Int
// }
// type BambooHfConfig struct {
// 	BlockNum      uint64
// 	Redelegations []BambooRedelegation
// }

type HardforksConfig struct {
	FixRedelegateBlockNum        uint64
	Redelegations                []Redelegation
	RewardsDistributionFrequency map[uint64]uint32
	MagnoliaHf                   MagnoliaHfConfig
	PhalaenopsisHfBlockNum       uint64
	FixClaimAllBlockNum          uint64
	AspenHf                      AspenHfConfig
	FicusHf                      FicusHfConfig
	CornusHf                     CornusHfConfig
	SoleiroliaHf                 SoleiroliaHfConfig
}

func (c *HardforksConfig) IsOnFixClaimAllHardfork(block types.BlockNum) bool {
	return block >= c.FixClaimAllBlockNum
}

func (c *HardforksConfig) IsOnPhalaenopsisHardfork(block types.BlockNum) bool {
	return block >= c.PhalaenopsisHfBlockNum
}

func (c *HardforksConfig) IsOnMagnoliaHardfork(block types.BlockNum) bool {
	return block >= c.MagnoliaHf.BlockNum
}

func (c *HardforksConfig) IsOnAspenHardforkPartOne(block types.BlockNum) bool {
	return block >= c.AspenHf.BlockNumPartOne
}

func (c *HardforksConfig) IsOnAspenHardforkPartTwo(block types.BlockNum) bool {
	return block >= c.AspenHf.BlockNumPartTwo
}

func (c *HardforksConfig) IsOnFicusHardfork(block types.BlockNum) bool {
	return block >= c.FicusHf.BlockNum
}

func (c *HardforksConfig) IsOnCornusHardfork(block types.BlockNum) bool {
	return block >= c.CornusHf.BlockNum
}

func (c *HardforksConfig) IsCornusHardfork(block types.BlockNum) bool {
	return block == c.CornusHf.BlockNum
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

func (c *HardforksConfig) Rules(num types.BlockNum) vm.Rules {
	return vm.Rules{
		IsMagnolia:     isForked(c.MagnoliaHf.BlockNum, num),
		IsAspenPartOne: isForked(c.AspenHf.BlockNumPartOne, num),
		IsAspenPartTwo: isForked(c.AspenHf.BlockNumPartTwo, num),
		IsFicus:        isForked(c.FicusHf.BlockNum, num),
		IsCornus:       isForked(c.CornusHf.BlockNum, num),
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
