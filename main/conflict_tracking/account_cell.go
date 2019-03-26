package conflict_tracking

import "github.com/Taraxa-project/taraxa-evm/common"

type AccountCell struct {
	memory map[common.Hash]*MemoryCell
	reads  map[TxId]DUMMY
	writes map[TxId]DUMMY
}

func (this *AccountCell) Constructor() *AccountCell {
	this.memory = make(map[common.Hash]*MemoryCell)
	this.reads = make(map[TxId]DUMMY)
	this.writes = make(map[TxId]DUMMY)
	return this
}

func (this *AccountCell) GetMemory(addr common.Hash) *MemoryCell {
	if cell, present := this.memory[addr]; present {
		return cell;
	}
	cell := new(MemoryCell).Init()
	this.memory[addr] = cell
	return cell
}
