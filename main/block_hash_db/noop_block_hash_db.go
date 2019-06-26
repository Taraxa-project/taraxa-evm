package block_hash_db

import "github.com/Taraxa-project/taraxa-evm/common"

type NotImplementedBlockHashStore struct{}

func (this *NotImplementedBlockHashStore) GetHeaderHashByBlockNumber(blockNumber uint64) common.Hash {
	panic("Not Implemented")
}
