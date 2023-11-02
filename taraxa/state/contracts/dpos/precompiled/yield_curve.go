package dpos

import (
	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
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

func (self *YieldCurve) CalculateBlockReward(current_total_delegation, current_total_tara_supply *uint256.Int) (block_reward *uint256.Int, yield *uint256.Int) {
	yield = self.calculateCurrentYield(current_total_tara_supply)
	block_reward = new(uint256.Int).Mul(current_total_delegation, yield)
	block_reward.Div(block_reward, new(uint256.Int).Mul(YieldFractionDecimalPrecision, uint256.NewInt(uint64(self.cfg.DPOS.BlocksPerYear))))

	return
}

// Calculates total supply based on genesis balances + rewards until Aspen hardfork
func (self *YieldCurve) CalculateTotalSupply(dpos_reader Reader) *uint256.Int {
	genesis_balances_sum := self.cfg.GenesisBalancesSum()
	total_supply, overflow := uint256.FromBig(genesis_balances_sum)
	asserts.Holds(overflow == false, "CalculateTotalSupply: Genesis balances sum oveflow")

	yield := uint256.NewInt(uint64(self.cfg.DPOS.YieldPercentage))
	// * 100 is here because yield is in %
	block_n_reward_divisor := new(uint256.Int).Mul(uint256.NewInt(100), uint256.NewInt(uint64(self.cfg.DPOS.BlocksPerYear)))

	for block_n := uint64(1); block_n < self.cfg.Hardforks.AspenHf.BlockNum; block_n++ {
		block_n_total_delegation := dpos_reader.TotalAmountDelegatedForBlock(block_n)
		asserts.Holds(block_n_total_delegation != nil, "CalculateTotalSupply: Unable to get total delegation")

		block_n_reward := new(uint256.Int).Mul(block_n_total_delegation, yield)
		block_n_reward.Div(block_n_reward, block_n_reward_divisor)

		total_supply.Add(total_supply, block_n_reward)
	}

	return total_supply
}
