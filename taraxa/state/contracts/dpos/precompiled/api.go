package dpos

import (
	"fmt"
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type DposConfigWithBlock struct {
	DposConfig chain_config.DPOSConfig
	Blk_n      types.BlockNum
}

type API struct {
	config_by_block []DposConfigWithBlock
	config          chain_config.ChainConfig
}

type GenesisTransfer = struct {
	Beneficiary common.Address
	Value       *big.Int
}

func (api *API) Init(cfg chain_config.ChainConfig) *API {
	asserts.Holds(cfg.DPOS.DelegationDelay <= cfg.DPOS.DelegationLockingPeriod)
	asserts.Holds(cfg.DPOS.DelegationDelay <= cfg.Hardforks.CornusHf.DelegationLockingPeriod)
	asserts.Holds(cfg.DPOS.DelegationDelay <= cfg.Hardforks.CactiHf.DelegationLockingPeriod)

	asserts.Holds(cfg.DPOS.EligibilityBalanceThreshold != nil)
	asserts.Holds(cfg.DPOS.VoteEligibilityBalanceStep != nil)
	asserts.Holds(cfg.DPOS.ValidatorMaximumStake != nil)
	asserts.Holds(cfg.DPOS.MinimumDeposit != nil)

	// MinimumDeposit must be <= ValidatorMaximumStake
	asserts.Holds(cfg.DPOS.ValidatorMaximumStake.Cmp(cfg.DPOS.MinimumDeposit) != -1)

	// ValidatorMaximumStake must be:
	//     > 0 as it is used for certain calculations in dpos contract, which require it to be != 0
	//     ValidatorMaximumStake * theoretical_max_reward_pool cannot overflow unit256
	asserts.Holds(cfg.DPOS.ValidatorMaximumStake.Cmp(big.NewInt(0)) == 1)
	// max uint256 == 2^256 == *10^77. Let ValidatorMaximumStake be half of that -> 10^38
	num_1e38 := big.NewInt(0)
	num_1e38.SetString("4B3B4CA85A86C47A098A224000000000", 16) // 10^38
	asserts.Holds(cfg.DPOS.ValidatorMaximumStake.Cmp(num_1e38) == -1)

	//MaxBlockAuthorReward is in %
	asserts.Holds(cfg.DPOS.MaxBlockAuthorReward <= 100)

	asserts.Holds(cfg.Hardforks.AspenHf.BlockNumPartTwo >= cfg.Hardforks.AspenHf.BlockNumPartOne)

	// total supply mus be <= max supply
	total_supply := cfg.GenesisBalancesSum()
	total_supply.Add(total_supply, cfg.Hardforks.AspenHf.GeneratedRewards)
	asserts.Holds(cfg.Hardforks.AspenHf.MaxSupply.Cmp(total_supply) >= 0, fmt.Sprintf("Hardforks.AspenHf.PartTwo.MaxSupply - Hardforks.AspenHf.PartTwo.GeneratedRewards (%d) must be >= Sum of genesis balances (%d)", cfg.Hardforks.AspenHf.MaxSupply, total_supply))

	total_supply_uin256, overflow := uint256.FromBig(total_supply)
	asserts.Holds(overflow == false, "total_supply overflow")

	var yield_curve YieldCurve
	yield_curve.Init(cfg)
	_, yield := yield_curve.CalculateBlockReward(uint256.NewInt(uint64(cfg.DPOS.BlocksPerYear)), uint256.NewInt(0), total_supply_uin256)

	decimal_precision := uint256.NewInt(1e4)
	ideal_yield := new(uint256.Int).Mul(uint256.NewInt(20), decimal_precision) // 20%

	if yield.Cmp(ideal_yield) == 1 {
		fmt.Printf("! Warning: Starting yield is %d %%, which is > ideal yield (20 %%). To make the yield some specific number, adjust either GenesisBalances or Hardforks.AspenHf.MaxSupply as Yield = (MaxSupply - Sum of GenesisBalances) / Sum of GenesisBalances\n", new(uint256.Int).Div(yield, decimal_precision))
	}

	api.config = cfg
	return api
}

func (api *API) GetConfigByBlockNum(blk_n uint64) chain_config.ChainConfig {
	for i, e := range api.config_by_block {
		// numeric_limits::max
		next_block_num := ^uint64(0)
		l_size := len(api.config_by_block)
		if i < l_size-1 {
			next_block_num = api.config_by_block[i+1].Blk_n
		}
		if (e.Blk_n <= blk_n) && (next_block_num > blk_n) {
			cfg := api.config
			cfg.DPOS = e.DposConfig
			return cfg
		}
	}
	return api.config
}

func (api *API) UpdateConfig(blk_n types.BlockNum, cfg chain_config.ChainConfig) {
	api.config_by_block = append(api.config_by_block, DposConfigWithBlock{cfg.DPOS, blk_n})
	api.config = cfg
}

func (api *API) NewContract(storage contract_storage.Storage, reader Reader, evm *vm.EVM) *Contract {
	return new(Contract).Init(api.config, storage, reader, evm)
}

func (api *API) NewSlashingContract(storage contract_storage.Storage, reader slashing.Reader, evm *vm.EVM) *slashing.Contract {
	return new(slashing.Contract).Init(api.config, storage, reader, evm)
}

func (api *API) InitAndRegisterAllContracts(storage contract_storage.Storage, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader, evm *vm.EVM, registry func(*common.Address, vm.PrecompiledContract)) {
	new(Contract).Init(api.config, storage, api.NewDelayedReader(blk_n, storage_factory), evm).Register(registry)
	if api.config.Hardforks.IsOnMagnoliaHardfork(blk_n) {
		new(slashing.Contract).Init(api.config, storage, api.NewSlashingReader(blk_n, storage_factory), evm).Register(registry)
	}
}

func (api *API) NewDelayedReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret Reader) {
	cfg := api.GetConfigByBlockNum(blk_n)
	ret.InitDelayedReader(&cfg, blk_n, storage_factory)
	return
}

func (api *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret Reader) {
	cfg := api.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}

func (api *API) NewSlashingReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret slashing.Reader) {
	cfg := api.GetConfigByBlockNum(blk_n)
	dpos_reader := api.NewDelayedReader(blk_n, storage_factory)
	ret.Init(&cfg, blk_n, dpos_reader, storage_factory)
	return
}
