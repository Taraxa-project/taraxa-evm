package dpos

import (
	"math/big"

	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/holiman/uint256"
)

// Yield curve description -> https://github.com/Taraxa-project/TIP/blob/main/TIP-2/TIP-2%20-%20Cap%20TARA's%20Total%20Supply.md

// Max token supply is 12 Billion TARA -> 12e+9(12 billion) * 1e+18(tara precision)
var maxTaraSupply = new(uint256.Int).Mul(uint256.NewInt(12e+9), uint256.NewInt(1e+18))

// Yield is calculated with 6 decimal precision
var YieldDecimalPrecision = uint256.NewInt(1e+6)

type YieldCurve struct {
	cfg chain_config.ChainConfig
}

func (self *YieldCurve) Init(cfg chain_config.ChainConfig) {
	self.cfg = cfg
}

// Note: yield is calculated with <yieldDecimalPrecision> decimal precision. To get % it must be divied by yieldDecimalPrecision
func (self *YieldCurve) calculateCurrentYield(current_total_delegation, current_total_tara_supply *uint256.Int) *uint256.Int {
	// Current yield = (max supply - current total supply) / current total supply
	current_yield := new(uint256.Int).Sub(maxTaraSupply, current_total_tara_supply)
	current_yield.Mul(current_yield, YieldDecimalPrecision)
	current_yield.Div(current_yield, current_total_tara_supply)

	return current_yield
}

func (self *YieldCurve) CalculateBlockReward(current_total_delegation, current_total_tara_supply *uint256.Int) (block_reward *uint256.Int, yield *uint256.Int) {
	yield = self.calculateCurrentYield(current_total_delegation, current_total_tara_supply)
	block_reward = new(uint256.Int).Mul(current_total_delegation, yield)
	block_reward.Div(block_reward, new(uint256.Int).Mul(new(uint256.Int).Mul(YieldDecimalPrecision, uint256.NewInt(100)), uint256.NewInt(uint64(self.cfg.DPOS.BlocksPerYear))))

	return
}

// Calculates total supply based on genesis balances + rewards until Aspen hardfork
func (self *YieldCurve) CalculateTotalSupply(dpos_reader Reader) (*uint256.Int, bool) {
	total_supply := big.NewInt(0)
	for _, balance := range self.cfg.GenesisBalances {
		total_supply.Add(total_supply, balance)
	}

	for block_n := uint64(0); block_n < self.cfg.Hardforks.AspenHfBlockNum; block_n++ {
		block_n_total_delegation := dpos_reader.TotalAmountDelegatedForBlock(block_n)

		block_n_reward := new(big.Int).Mul(block_n_total_delegation, big.NewInt(int64(self.cfg.DPOS.YieldPercentage)))
		block_n_reward.Div(block_n_reward, new(big.Int).Mul(big.NewInt(100), big.NewInt(int64(self.cfg.DPOS.BlocksPerYear))))

		total_supply.Add(total_supply, block_n_reward)
	}

	return uint256.FromBig(total_supply)
}
