package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type pending_account struct {
	acc         state_common.Account
	trie_w      *trie.Writer
	executor    util.SingleThreadExecutor
	enc_storage []byte
	enc_hash    []byte
}

func (self *pending_account) EncodeForTrie() (r0, r1 []byte) {
	self.executor.Synchronize()
	r0, r1 = self.enc_storage, self.enc_hash
	return
}
