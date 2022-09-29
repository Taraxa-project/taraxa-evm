package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

type FeesRewards struct {
	feesRewards map[common.Address]*big.Int
}

func (self *FeesRewards) Init() {
	self.feesRewards = make(map[common.Address]*big.Int)
}

func (self *FeesRewards) AddTxFeeReward(account common.Address, reward *big.Int) {
	feesReward, feesRewardExists := self.feesRewards[account]
	if feesRewardExists {
		self.feesRewards[account] = bigutil.Add(feesReward, reward)
	} else {
		self.feesRewards[account] = reward
	}
}

func (self *FeesRewards) GetTxsFeesReward(account common.Address) *big.Int {
	feesReward, feeRewardExists := self.feesRewards[account]
	if feeRewardExists {
		return feesReward
	} else {
		return big.NewInt(0)
	}
}

func NewFeesRewards() FeesRewards {
	feesRewards := FeesRewards{}
	feesRewards.Init()

	return feesRewards
}
