package conflict_detector

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type ConflictDetector struct {
	inbox             chan *Operation
	onConflict        OnConflict
	operationLog      operationLog
	keysInConflict    Keys
	authorsInConflict Authors
	executionMutex    sync.Mutex
	haltMutex         sync.Mutex
	halted            bool
}

func New(inboxCapacity int, onConflict OnConflict) *ConflictDetector {
	this := &ConflictDetector{
		inbox:      make(chan *Operation, util.Max(inboxCapacity, 1)),
		onConflict: onConflict,
	}
	for opType := range this.operationLog {
		this.operationLog[opType] = make(map[Key]Authors)
	}
	return this
}

func (this *ConflictDetector) Run() {
	defer concurrent.LockUnlock(&this.executionMutex)()
	this.authorsInConflict = linkedhashset.New()
	this.keysInConflict = linkedhashset.New()
	concurrent.WithLock(&this.haltMutex, func() {
		this.halted = false
	})
	for {
		if op := <-this.inbox; op != nil {
			this.process(op)
		} else {
			break
		}
	}
}

func (this *ConflictDetector) Halt() {
	defer concurrent.LockUnlock(&this.haltMutex)()
	if !this.halted {
		this.halted = true
		this.inbox <- nil
	}
}

func (this *ConflictDetector) AwaitResult() Authors {
	defer concurrent.LockUnlock(&this.executionMutex)()
	return this.authorsInConflict
}

// This function is thread safe
// The returned logger is not thread safe
func (this *ConflictDetector) NewLogger(author Author) OperationLogger {
	cache := make(map[OperationType]Keys)
	return func(opType OperationType, key Key) {
		cachedKeys := cache[opType]
		if cachedKeys == nil {
			cachedKeys = linkedhashset.New()
			cache[opType] = cachedKeys
		} else if cachedKeys.Contains(key) {
			return
		}
		defer concurrent.LockUnlock(&this.haltMutex)()
		util.Assert(!this.halted)
		this.inbox <- &Operation{author, opType, key}
		cachedKeys.Add(key)
	}
}

func (this *ConflictDetector) registerConflict(op *Operation, authors Authors) {
	authors.Each(func(_ int, author Author) {
		this.authorsInConflict.Add(author)
	})
	if this.onConflict != nil {
		// new goroutine to prevent deadlocking misuse
		go this.onConflict(op, authors)
	}
}

func (this *ConflictDetector) process(op *Operation) {
	if this.authorsInConflict.Contains(op.Author) {
		return
	}
	if this.keysInConflict.Contains(op.Key) {
		this.registerConflict(op, linkedhashset.New(op.Author))
		return
	}
	conflictFound := false
	for _, conflictingType := range conflictRelations[op.Type] {
		authors := this.operationLog[conflictingType][op.Key]
		if authors != nil && (authors.Size() > 1 || !authors.Contains(op.Author)) {
			conflictFound = true
			break
		}
	}
	if conflictFound {
		this.keysInConflict.Add(op.Key)
		for _, opsByKey := range this.operationLog {
			if authors := opsByKey[op.Key]; authors != nil {
				this.registerConflict(op, authors)
				delete(opsByKey, op.Key)
			}
		}
	} else {
		opsByKey := this.operationLog[op.Type]
		authors := opsByKey[op.Key]
		if authors == nil {
			authors = linkedhashset.New()
			opsByKey[op.Key] = authors
		}
		authors.Add(op.Author)
	}
}