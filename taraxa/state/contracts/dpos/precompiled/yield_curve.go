package dpos

import (
	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/holiman/uint256"
)

// Yield curve description -> https://github.com/Taraxa-project/TIP/blob/main/TIP-2/TIP-2%20-%20Cap%20TARA's%20Total%20Supply.md

// Yield is calculated with 6 decimal precision
var YieldFractionDecimalPrecision = uint256.NewInt(1e+6)

type YieldCurve struct {
	cfg        chain_config.ChainConfig
	max_supply *uint256.Int
}

func (self *YieldCurve) Init(cfg chain_config.ChainConfig) {
	self.cfg = cfg

	max_supply, overflow := uint256.FromBig(self.cfg.Hardforks.AspenHf.MaxSupply)
	asserts.Holds(overflow == false, "YieldCurve max supply overflow")
	self.max_supply = max_supply
}

// Note: yield is calculated with <YieldFractionDecimalPrecision> decimal precision. To get % it must be divied by YieldFractionDecimalPrecision
func (self *YieldCurve) calculateCurrentYield(current_total_tara_supply *uint256.Int) *uint256.Int {
	// Current yield = (max supply - current total supply) / current total supply
	current_yield := new(uint256.Int).Sub(self.max_supply, current_total_tara_supply)
	current_yield.Mul(current_yield, YieldFractionDecimalPrecision)
	current_yield.Div(current_yield, current_total_tara_supply)

	return current_yield
}

func (self *YieldCurve) CalculateBlockReward(blocks_per_year, current_total_delegation, current_total_tara_supply *uint256.Int) (block_reward *uint256.Int, yield *uint256.Int) {
	yield = self.calculateCurrentYield(current_total_tara_supply)
	block_reward = new(uint256.Int).Mul(current_total_delegation, yield)
	block_reward.Div(block_reward, new(uint256.Int).Mul(YieldFractionDecimalPrecision, blocks_per_year))
	return
}

// Calculates total supply based on minted_toknes + genesis balances + total generated rewards until Aspen hardfork
func (self *YieldCurve) CalculateTotalSupply(minted_tokens *uint256.Int) *uint256.Int {
	total_supply := bigutil.Add(self.cfg.GenesisBalancesSum(), minted_tokens.ToBig())
	total_supply.Add(total_supply, self.cfg.Hardforks.AspenHf.GeneratedRewards)

	total_supply_uint256, overflow := uint256.FromBig(total_supply)
	asserts.Holds(overflow == false, "CalculateTotalSupply: Genesis balances sum oveflow")

	return total_supply_uint256
}
