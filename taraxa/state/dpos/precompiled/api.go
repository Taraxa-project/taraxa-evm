package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

func ContractAddress() common.Address {
	return *contract_address
}

type API struct {
	cfg_by_block     []ConfigWithBlock
	dpos_config      chain_config.DposConfig
	hardforks_config chain_config.HardforksConfig
}

type ConfigWithBlock struct {
	cfg   chain_config.DposConfig
	blk_n types.BlockNum
}

type GenesisTransfer = struct {
	Beneficiary common.Address
	Value       *big.Int
}

func (self *API) Init(cfg chain_config.DposConfig, hardforks chain_config.HardforksConfig) *API {
	asserts.Holds(cfg.DelegationDelay <= cfg.DelegationLockingPeriod)

	asserts.Holds(cfg.EligibilityBalanceThreshold != nil)
	asserts.Holds(cfg.VoteEligibilityBalanceStep != nil)
	asserts.Holds(cfg.ValidatorMaximumStake != nil)
	asserts.Holds(cfg.MinimumDeposit != nil)

	// MinimumDeposit must be <= ValidatorMaximumStake
	asserts.Holds(cfg.ValidatorMaximumStake.Cmp(cfg.MinimumDeposit) != -1)

	// ValidatorMaximumStake must be:
	//     > 0 as it is used for certain calculations in dpos contract, which require it to be != 0
	//     ValidatorMaximumStake * theoretical_max_reward_pool cannot overflow unit256
	asserts.Holds(cfg.ValidatorMaximumStake.Cmp(big.NewInt(0)) == 1)
	// max uint256 == 2^256 == *10^77. Let ValidatorMaximumStake be half of that -> 10^38
	num_1e38 := big.NewInt(0)
	num_1e38.SetString("4B3B4CA85A86C47A098A224000000000", 16) // 10^38
	asserts.Holds(cfg.ValidatorMaximumStake.Cmp(num_1e38) == -1)

	//MaxBlockAuthorReward is in %
	asserts.Holds(cfg.MaxBlockAuthorReward <= 100)

	self.dpos_config = cfg
	self.hardforks_config = hardforks
	return self
}

func (self *API) GetConfigByBlockNum(blk_n uint64) chain_config.DposConfig {
	for i, e := range self.cfg_by_block {
		// numeric_limits::max
		next_block_num := ^uint64(0)
		l_size := len(self.cfg_by_block)
		if i < l_size-1 {
			next_block_num = self.cfg_by_block[i+1].blk_n
		}
		if (e.blk_n <= blk_n) && (next_block_num > blk_n) {
			return e.cfg
		}
	}
	return self.dpos_config
}

func (self *API) UpdateConfig(blk_n types.BlockNum, cfg chain_config.DposConfig) {
	self.cfg_by_block = append(self.cfg_by_block, ConfigWithBlock{cfg, blk_n})
	self.dpos_config = cfg
}

func (self *API) NewContract(storage Storage, reader Reader, evm *vm.EVM) *Contract {
	return new(Contract).Init(self.dpos_config, self.hardforks_config, storage, reader, evm)
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) (ret Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}
