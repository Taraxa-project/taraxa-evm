package conflict_tracking

import "github.com/Taraxa-project/taraxa-evm/common"

type MemoryCell struct {
	reads  map[TxId]common.Hash
	writes map[TxId]common.Hash
}

func (this *MemoryCell) Init() *MemoryCell {
	this.reads = make(map[TxId]common.Hash)
	this.writes = make(map[TxId]common.Hash)
	return this
}

func (this *MemoryCell) Read(id TxId) {

}

func (this *MemoryCell) Write(id TxId) {

}
