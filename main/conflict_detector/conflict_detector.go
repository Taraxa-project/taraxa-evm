package conflict_detector

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type operationType int

const (
	GET operationType = iota
	SET
	ADD
)

type conflictRelationsMap = map[operationType][]operationType

var conflictRelations = func() conflictRelationsMap {
	ret := make(conflictRelationsMap)
	inConflict := func(left, right operationType) {
		ret[left] = append(ret[left], right)
		ret[right] = append(ret[right], left)
	}
	inConflict(GET, SET)
	inConflict(GET, ADD)
	inConflict(SET, SET)
	inConflict(SET, ADD)
	return ret
}()

type Author = interface{} // equals/hashcode required
type Authors = *linkedhashset.Set
type Key = string // equals/hashcode required
type Keys = *linkedhashset.Set

type operation struct {
	Author Author
	Type   operationType
	Key    Key
}

type ops = map[operationType]map[Key]Authors

type Logger func(operationType, Key)

type ConflictDetector struct {
	inbox             chan *operation
	ops               ops
	keysInConflict    Keys
	authorsInConflict Authors
	conflictMutex     sync.RWMutex
	executionMutex    sync.Mutex
}

func (this *ConflictDetector) Init(inboxCapacity uint64) *ConflictDetector {
	this.ops = make(ops)
	for opType := GET; opType <= ADD; opType++ {
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

func (this *ConflictDetector) GetLogger(author Author) Logger {
	cache := make(map[operationType]Keys)
	return func(t operationType, k Key) {
		cachedKeys := cache[t]
		if cachedKeys == nil {
			cachedKeys = linkedhashset.New()
			cache[t] = cachedKeys
		} else if cachedKeys.Contains(k) {
			return
		}
		cachedKeys.Add(k)
		this.inbox <- &operation{
			Author: author,
			Key:    k,
			Type:   t,
		}
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
		return
	}
	opsByKey := this.ops[op.Type]
	authors := opsByKey[op.Key]
	if authors == nil {
		authors = linkedhashset.New()
		opsByKey[op.Key] = authors
	}
	authors.Add(op.Author)
}
