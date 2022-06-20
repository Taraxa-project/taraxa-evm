package rewards_stats

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

type ValidatorStats struct {
	// Unique transactions counter -> how many unique txs validator included in his dag blocks
	// Unique txs is what defines quality of block -> block with 10 unique transactions is 10 times more valuable
	// than block with single unique transaction.
	UniqueTxsCount uint32

	// Validator cert voted block
	ValidCertVote bool
}

type RewardsStats struct {
	// Validator stats
	ValidatorsStats map[common.Address]ValidatorStats

	// Total unique transactions counter
	TotalUniqueTxsCount uint32

	// Total unique votes counter
	TotalVotesCount uint32

	// Total count of addtional votes in block
	BonusVotesCount uint32
}

func NewRewardsStats() RewardsStats {
	rewardsStats := RewardsStats{}
	rewardsStats.ValidatorsStats = make(map[common.Address]ValidatorStats)

	return rewardsStats
}
