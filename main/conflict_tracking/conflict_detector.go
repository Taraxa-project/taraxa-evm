package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/main/util"
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
	Author  Author
	IsWrite bool
	Key     Key
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
	if op.IsWrite {
		this.processWrite(op.Author, op.Key)
	} else {
		this.processRead(op.Author, op.Key)
	}
}

func (this *ConflictDetector) processWrite(author Author, key Key) {
	reads := this.reads[key]
	prevWriter, hasBeenWrite := this.writes[key]
	writeConflict := hasBeenWrite && prevWriter != author
	haveBeenReads := reads != nil && reads.Size() > 0
	readConflict := haveBeenReads && !(reads.Size() == 1 && reads.Contains(author))
	if (readConflict || writeConflict) {
		this.keysInConflict.Add(key)
		this.addConflictingAuthor(author)
		if writeConflict {
			this.addConflictingAuthor(prevWriter)
		}
		if haveBeenReads {
			reads.Each(func(index int, value Author) {
				this.addConflictingAuthor(value)
			})
		}
		delete(this.writes, key)
		delete(this.reads, key)
	} else if !hasBeenWrite {
		this.writes[key] = author
	}
}

func (this *ConflictDetector) processRead(author Author, key Key) {
	reads := this.reads[key]
	if reads == nil {
		reads = linkedhashset.New()
		this.reads[key] = reads
	}
	reads.Add(author)
}
