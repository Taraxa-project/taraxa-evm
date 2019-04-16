package conflict_detector

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type OperationType int

const (
	GET OperationType = iota
	SET
	ADD            //commutative
	// TODO consider operands of the operators
	DEFAULT_INITIALIZE  //idempotent
	OperationType_count uint = iota
)

type conflictRelationsMap = map[OperationType][]OperationType

var conflictRelations = func() conflictRelationsMap {
	ret := make(conflictRelationsMap)
	inConflict := func(left, right OperationType) {
		ret[left] = append(ret[left], right)
		ret[right] = append(ret[right], left)
	}
	inConflict(GET, SET)
	inConflict(GET, ADD)
	inConflict(SET, SET)
	inConflict(SET, ADD)
	inConflict(DEFAULT_INITIALIZE, GET)
	inConflict(DEFAULT_INITIALIZE, ADD)
	inConflict(DEFAULT_INITIALIZE, SET)
	return ret
}()

type Author = interface{} // equals/hashcode required
type Authors = *linkedhashset.Set
type Key = string // equals/hashcode required
type Keys = *linkedhashset.Set
type Logger func(OperationType, Key)

type operation struct {
	Author Author
	Type   OperationType
	Key    Key
}

type ops = []map[Key]Authors

type ConflictDetector struct {
	inbox             chan *operation
	ops               ops
	keysInConflict    Keys
	authorsInConflict Authors
	conflictMutex     sync.RWMutex
	executionMutex    sync.Mutex
}

func (this *ConflictDetector) Init(inboxCapacity int) *ConflictDetector {
	this.ops = make(ops, OperationType_count)
	for opType := range this.ops {
		this.ops[opType] = make(map[Key]Authors)
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

// The returned logger is not thread safe
func (this *ConflictDetector) NewLogger(author Author) Logger {
	cache := make(map[OperationType]Keys)
	return func(t OperationType, k Key) {
		cachedKeys := cache[t]
		if cachedKeys == nil {
			cachedKeys = linkedhashset.New()
			cache[t] = cachedKeys
		} else if cachedKeys.Contains(k) {
			return
		}
		this.inbox <- &operation{
			Author: author,
			Key:    k,
			Type:   t,
		}
		cachedKeys.Add(k)
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
		authors := this.ops[conflictingType][op.Key]
		if authors != nil && (authors.Size() > 1 || !authors.Contains(op.Author)) {
			conflictFound = true
			break
		}
	}
	if conflictFound {
		this.keysInConflict.Add(op.Key)
		for _, opsByKey := range this.ops {
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
		opsByKey := this.ops[op.Type]
		authors := opsByKey[op.Key]
		if authors == nil {
			authors = linkedhashset.New()
			opsByKey[op.Key] = authors
		}
		authors.Add(op.Author)
	}
}
