package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

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

func (self *API) Init(cfg chain_config.ChainConfig) *API {
	asserts.Holds(cfg.DPOS.DelegationDelay <= cfg.DPOS.DelegationLockingPeriod)

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

	self.config = cfg
	return self
}

func (self *API) GetConfigByBlockNum(blk_n uint64) chain_config.ChainConfig {
	for i, e := range self.config_by_block {
		// numeric_limits::max
		next_block_num := ^uint64(0)
		l_size := len(self.config_by_block)
		if i < l_size-1 {
			next_block_num = self.config_by_block[i+1].Blk_n
		}
		if (e.Blk_n <= blk_n) && (next_block_num > blk_n) {
			cfg := self.config
			cfg.DPOS = e.DposConfig
			return cfg
		}
	}
	return self.config
}

func (self *API) UpdateConfig(blk_n types.BlockNum, cfg chain_config.ChainConfig) {
	self.config_by_block = append(self.config_by_block, DposConfigWithBlock{cfg.DPOS, blk_n})
	self.config = cfg
}

func (self *API) NewContract(storage contract_storage.Storage, reader Reader, evm *vm.EVM) *Contract {
	return new(Contract).Init(self.config, storage, reader, evm)
}

func (self *API) NewSlashingContract(storage contract_storage.Storage, reader slashing.Reader, evm *vm.EVM) *slashing.Contract {
	return new(slashing.Contract).Init(self.config, storage, reader, evm)
}

func (self *API) InitAndRegisterAllContracts(storage contract_storage.Storage, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader, evm *vm.EVM, registry func(*common.Address, vm.PrecompiledContract)) {
	new(Contract).Init(self.config, storage, self.NewReader(blk_n, storage_factory), evm).Register(registry)
	if self.config.Hardforks.IsMagnoliaHardfork(blk_n) {
		new(slashing.Contract).Init(self.config, storage, self.NewSlashingReader(blk_n, storage_factory), evm).Register(registry)
	}
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}

func (self *API) NewSlashingReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret slashing.Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	dpos_reader := self.NewReader(blk_n, storage_factory)
	ret.Init(&cfg, blk_n, dpos_reader, storage_factory)
	return
}
