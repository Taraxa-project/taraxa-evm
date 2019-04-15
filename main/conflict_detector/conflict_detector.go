package conflict_detector

import (
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/collections"
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type Author = interface{} // equals/hashcode required
type Authors = *linkedhashset.Set
type Key = string // equals/hashcode required
type Keys = *linkedhashset.Set

type ConflictDetector struct {
	inbox             chan *Operation
	reads             map[Key]Authors
	writes            map[Key]Author
	keysInConflict    Keys
	authorsInConflict Authors
	conflictMutex     sync.RWMutex
	executionMutex    sync.Mutex
}

type Operation struct {
	Author            Author
	IsWrite           bool
	DontRaiseConflict bool
	Key               Key
}

func (this *ConflictDetector) Init(inboxCapacity uint64) *ConflictDetector {
	this.inbox = make(chan *Operation, inboxCapacity)
	this.reads = make(map[Key]Authors)
	this.writes = make(map[Key]Author)
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
	//this.inbox = make(chan *Operation, cap(this.inbox))
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

func (this *ConflictDetector) Submit(op *Operation) {
	util.Assert(op != nil)
	this.inbox <- op
	//this.process(op)
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

func (this *ConflictDetector) process(op *Operation) {
	if this.IsCurrentlyInConflict(op.Author) {
		return
	}
	if this.keysInConflict.Contains(op.Key) {
		this.addConflictingAuthor(op.Author)
		return
	}
	key, author, isWrite := op.Key, op.Author, op.IsWrite
	reads := this.reads[key]
	haveBeenReads := !collections.IsEmpty(reads)
	if !op.DontRaiseConflict {
		prevWriter, hasBeenWrite := this.writes[key]
		conflictFound := hasBeenWrite && prevWriter != author ||
			isWrite && haveBeenReads && !collections.ContainsExactly(reads, author)
		if conflictFound {
			this.keysInConflict.Add(key)
			this.addConflictingAuthor(author)
			if hasBeenWrite {
				this.addConflictingAuthor(prevWriter)
				delete(this.writes, key)
			}
			if haveBeenReads {
				reads.Each(func(index int, reader Author) {
					this.addConflictingAuthor(reader)
				})
				delete(this.reads, key)
			}
			return
		}
	}
	if isWrite {
		this.writes[key] = author
	} else {
		if !haveBeenReads {
			reads = linkedhashset.New()
			this.reads[key] = reads
		}
		reads.Add(author)
	}
}