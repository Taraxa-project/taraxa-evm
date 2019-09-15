package conflict_detector

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"sync/atomic"
)

type OnConflict = func(op *Operation, conflicts *AuthorsByOperation, hasCaused bool)
type OperationLogger = func(OperationType, Key)

type status = uint32

const (
	runnable status = iota
	running
	stopped_forcibly
)

type ConflictDetectorActor struct {
	conflictDetector *ConflictDetector
	onConflict       OnConflict
	inbox            chan *Operation
	quitChan         chan byte
	status           status
}

func NewConflictDetectorActor(detector *ConflictDetector, inboxCapacity int, onConflict OnConflict) *ConflictDetectorActor {
	return &ConflictDetectorActor{
		conflictDetector: detector,
		onConflict:       onConflict,
		inbox:            make(chan *Operation, util.Max(inboxCapacity, 1)),
		quitChan:         make(chan byte),
	}
}

func (this *ConflictDetectorActor) Run() {
	util.Assert(atomic.CompareAndSwapUint32(&this.status, runnable, running))
	defer close(this.quitChan)
	for {
		op := <-this.inbox
		if op == nil || atomic.LoadUint32(&this.status) == stopped_forcibly {
			break
		}
		if conflicts, hasCaused := this.conflictDetector.Process(op); conflicts != nil {
			this.onConflict(op, conflicts, hasCaused)
		}
	}
}

func (this *ConflictDetectorActor) ForceShutdown() *ConflictDetectorActor {
	atomic.StoreUint32(&this.status, stopped_forcibly)
	return this
}

func (this *ConflictDetectorActor) SendTerminator() *ConflictDetectorActor {
	this.inbox <- nil
	return this
}

func (this *ConflictDetectorActor) Await() *ConflictDetector {
	<-this.quitChan
	return this.conflictDetector
}

// This function is thread safe
// The returned logger is not thread safe
func (this *ConflictDetectorActor) NewOperationLogger(author Author) OperationLogger {
	var cache [OperationType_count]Keys
	return func(opType OperationType, key Key) {
		cachedKeys := cache[opType]
		if cachedKeys == nil {
			cachedKeys = make(Keys)
			cache[opType] = cachedKeys
		} else if cachedKeys[key] {
			return
		}
		this.inbox <- &Operation{author, opType, key}
		cachedKeys[key] = true
	}
}
