package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type ConflictDetector struct {
	inbox                 chan *operation
	reads                 map[string]*linkedhashset.Set
	writes                map[string]TxId
	currentKeysInConflict *linkedhashset.Set
	currentConflictingTx  *linkedhashset.Set
	oldConflictingTx      *linkedhashset.Set
	conflictMutex         sync.RWMutex
	executionMutex        sync.Mutex
	isRefusingWrites      uint32
}

type operation struct {
	txId    TxId
	isWrite bool
	account common.Address
	key     string
}

func (this *ConflictDetector) Init(inboxCapacity uint64) *ConflictDetector {
	this.inbox = make(chan *operation, inboxCapacity)
	this.reads = make(map[string]*linkedhashset.Set)
	this.writes = make(map[string]TxId)
	this.currentKeysInConflict = linkedhashset.New()
	this.currentConflictingTx = linkedhashset.New()
	this.oldConflictingTx = linkedhashset.New()
	return this
}

func (this *ConflictDetector) Run() {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	for {
		op := <-this.inbox
		if op == nil {
			break
		}
		this.process(op)
	}
	//close(this.inbox)
	//this.inbox = make(chan *operation, cap(this.inbox))
}

func (this *ConflictDetector) SignalShutdown() *ConflictDetector {
	this.inbox <- nil
	return this
}

func (this *ConflictDetector) Join() *ConflictDetector {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	return this
}

func (this *ConflictDetector) Reset(conflictReceiver func(id TxId)) {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	this.currentConflictingTx.Each(func(index int, value interface{}) {
		txId := value.(TxId)
		this.oldConflictingTx.Add(txId)
		conflictReceiver(txId)
	})
	this.currentConflictingTx.Clear()
	this.currentKeysInConflict.Clear()
}

func (this *ConflictDetector) Submit(op *operation) {
	util.Assert(op != nil)
	this.inbox <- op
}

func (this *ConflictDetector) IsCurrentlyInConflict(id TxId) bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return this.currentConflictingTx.Contains(id)
}

func (this *ConflictDetector) HaveBeenConflicts() bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return !this.currentConflictingTx.Empty()
}

func (this *ConflictDetector) process(op *operation) {
	if this.oldConflictingTx.Contains(op.txId) || this.IsCurrentlyInConflict(op.txId) {
		return
	}
	accountKey := op.account.Hex()
	fullKey := accountKey + op.key
	keys := []string{accountKey, fullKey}
	for _, key := range keys {
		if !this.checkKey(op.txId, key) {
			return
		}
	}
	for _, key := range keys {
		if op.isWrite {
			this.processWrite(op.txId, key)
		} else {
			this.processRead(op.txId, key)
		}
	}
}

func (this *ConflictDetector) checkKey(id TxId, key string) (conflictFound bool) {
	if this.currentKeysInConflict.Contains(key) {
		this.addConflictingTx(id)
		conflictFound = true
	}
	return
}

func (this *ConflictDetector) processWrite(txId TxId, key string) (conflictFound bool) {
	reads := this.reads[key]
	prevWriter, hasBeenWrite := this.writes[key]
	writeConflict := hasBeenWrite && prevWriter != txId
	haveBeenReads := reads != nil && reads.Size() > 0
	readConflict := haveBeenReads && !(reads.Size() == 1 && reads.Contains(txId))
	if (readConflict || writeConflict) {
		this.currentKeysInConflict.Add(key)
		this.addConflictingTx(txId)
		if writeConflict {
			this.addConflictingTx(prevWriter)
		}
		if haveBeenReads {
			reads.Each(func(index int, value interface{}) {
				this.addConflictingTx(value.(TxId))
			})
		}
		delete(this.writes, key)
		delete(this.reads, key)
		conflictFound = true
	} else if !hasBeenWrite {
		this.writes[key] = txId
	}
	return
}

func (this *ConflictDetector) processRead(txId TxId, key string) {
	reads := this.reads[key]
	if reads == nil {
		reads = linkedhashset.New()
		this.reads[key] = reads
	}
	reads.Add(txId)
}

func (this *ConflictDetector) addConflictingTx(id TxId) {
	this.conflictMutex.Lock()
	defer this.conflictMutex.Unlock()
	this.currentConflictingTx.Add(id)
}
