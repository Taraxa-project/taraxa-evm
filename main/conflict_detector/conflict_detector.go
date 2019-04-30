package conflict_detector

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type ConflictDetector struct {
	inbox             chan *operation
	operationLog      operationLog
	keysInConflict    Keys
	authorsInConflict Authors
	conflictMutex     sync.RWMutex
	executionMutex    sync.Mutex
}

func New(inboxCapacity int) *ConflictDetector {
	this := new(ConflictDetector)
	this.operationLog = make(operationLog, OperationType_count)
	for opType := range this.operationLog {
		this.operationLog[opType] = make(map[Key]Authors)
	}
	this.inbox = make(chan *operation, inboxCapacity)
	this.Reset()
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
	// TODO close channel?
	//close(this.inbox)
	//this.inbox = make(chan *operation, cap(this.inbox))
}

func (this *ConflictDetector) RequestShutdown() *ConflictDetector {
	this.inbox <- nil
	return this
}

func (this *ConflictDetector) Join() *ConflictDetector {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	return this
}

// This function is thread safe
// The returned logger is not thread safe
func (this *ConflictDetector) NewLogger(author Author) Logger {
	cache := make(map[OperationType]Keys)
	return func(opType OperationType, key Key) {
		cachedKeys := cache[opType]
		if cachedKeys == nil {
			cachedKeys = linkedhashset.New()
			cache[opType] = cachedKeys
		} else if cachedKeys.Contains(key) {
			return
		}
		this.inbox <- &operation{
			Author: author,
			Key:    key,
			Type:   opType,
		}
		cachedKeys.Add(key)
	}
}

func (this *ConflictDetector) Reset() (authorsInConflict Authors) {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	this.conflictMutex.Lock()
	defer this.conflictMutex.Unlock()
	authorsInConflict = this.authorsInConflict
	this.authorsInConflict = linkedhashset.New()
	this.keysInConflict = linkedhashset.New()
	return
}

func (this *ConflictDetector) IsCurrentlyInConflict(author Author) bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return this.authorsInConflict.Contains(author)
}

func (this *ConflictDetector) HaveBeenConflicts() bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return !this.authorsInConflict.Empty()
}

func (this *ConflictDetector) addConflictingAuthor(author Author) {
	this.conflictMutex.Lock()
	defer this.conflictMutex.Unlock()
	this.authorsInConflict.Add(author)
}

func (this *ConflictDetector) process(op *operation) {
	if this.IsCurrentlyInConflict(op.Author) {
		return
	}
	if this.keysInConflict.Contains(op.Key) {
		this.addConflictingAuthor(op.Author)
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
			authors := opsByKey[op.Key]
			if authors == nil {
				continue
			}
			authors.Each(func(index int, author Author) {
				this.addConflictingAuthor(author)
			})
			delete(opsByKey, op.Key)
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
