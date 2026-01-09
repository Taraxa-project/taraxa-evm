package rewards_stats

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type ValidatorStats struct {
	// Unique transactions counter -> how many unique txs validator included in his dag blocks
	// Unique txs is what defines quality of block -> block with 10 unique transactions is 10 times more valuable
	// than block with single unique transaction.
	DagBlocksCount uint32

	// Validator cert voted block weight
	VoteWeight uint64

	// Fee rewards
	FeesRewards *big.Int
}

type RewardsStats struct {
	// Pbft block author
	BlockAuthor common.Address

	// Expected number of blocks generated per year based on block's dynamic lambda
	BlocksPerYear uint32

	// Validator stats
	ValidatorsStats map[common.Address]ValidatorStats

	// Total unique transactions counter
	TotalDagBlocksCount uint32

	// Total weight of votes in block
	TotalVotesWeight uint64

	// Max weight of votes in block
	MaxVotesWeight uint64
}
