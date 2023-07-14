package dpos

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/holiman/uint256"
)

type FeesRewards struct {
	feesRewards map[common.Address]*uint256.Int
}

func (self *FeesRewards) Init() {
	self.feesRewards = make(map[common.Address]*uint256.Int)
}

func (self *FeesRewards) AddTrxFeeReward(account common.Address, reward *uint256.Int) {
	feesReward, feesRewardExists := self.feesRewards[account]
	if feesRewardExists {
		self.feesRewards[account].Add(feesReward, reward)
	} else {
		self.feesRewards[account] = reward
	}
}

func (self *FeesRewards) GetTrxsFeesReward(account common.Address) *uint256.Int {
	feesReward, feeRewardExists := self.feesRewards[account]
	if feeRewardExists {
		return feesReward
	} else {
		return uint256.NewInt(0)
	}
}

func NewFeesRewards() FeesRewards {
	feesRewards := FeesRewards{}
	feesRewards.Init()

	return feesRewards
}
