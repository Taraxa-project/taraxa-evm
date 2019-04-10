package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/emirpasic/gods/sets/linkedhashset"
	"sync"
)

type ConflictDetector struct {
	inbox                     chan *Operation
	reads                     map[string]*linkedhashset.Set
	writes                    map[string]interface{}
	keysInConflict            *linkedhashset.Set
	currentConflictingAuthors *linkedhashset.Set
	ignoredAuthors            *linkedhashset.Set
	conflictMutex             sync.RWMutex
	executionMutex            sync.Mutex
}

type Operation struct {
	Author  interface{}
	IsWrite bool
	Key     string
}

func (this *ConflictDetector) Init(inboxCapacity uint64) *ConflictDetector {
	this.inbox = make(chan *Operation, inboxCapacity)
	this.reads = make(map[string]*linkedhashset.Set)
	this.writes = make(map[string]interface{})
	this.keysInConflict = linkedhashset.New()
	this.currentConflictingAuthors = linkedhashset.New()
	this.ignoredAuthors = linkedhashset.New()
	return this
}

func (this *ConflictDetector) IgnoreAuthor(author interface{}) *ConflictDetector {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	this.ignoredAuthors.Add(author)
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

func (this *ConflictDetector) Reset(conflictReceiver func(author interface{})) {
	this.executionMutex.Lock()
	defer this.executionMutex.Unlock()
	this.currentConflictingAuthors.Each(func(index int, author interface{}) {
		this.ignoredAuthors.Add(author)
		conflictReceiver(author)
	})
	this.currentConflictingAuthors.Clear()
	this.keysInConflict.Clear()
}

func (this *ConflictDetector) Submit(op *Operation) {
	util.Assert(op != nil)
	this.inbox <- op
	//this.process(op)
}

func (this *ConflictDetector) IsCurrentlyInConflict(author interface{}) bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return this.currentConflictingAuthors.Contains(author)
}

func (this *ConflictDetector) HaveBeenConflicts() bool {
	this.conflictMutex.RLock()
	defer this.conflictMutex.RUnlock()
	return !this.currentConflictingAuthors.Empty()
}

func (this *ConflictDetector) process(op *Operation) {
	if this.ignoredAuthors.Contains(op.Author) || this.IsCurrentlyInConflict(op.Author) {
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

func (this *ConflictDetector) processWrite(author interface{}, key string) {
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
			reads.Each(func(index int, value interface{}) {
				this.addConflictingAuthor(value)
			})
		}
		delete(this.writes, key)
		delete(this.reads, key)
	} else if !hasBeenWrite {
		this.writes[key] = author
	}
}

func (this *ConflictDetector) processRead(author interface{}, key string) {
	reads := this.reads[key]
	if reads == nil {
		reads = linkedhashset.New()
		this.reads[key] = reads
	}
	reads.Add(author)
}

func (this *ConflictDetector) addConflictingAuthor(author interface{}) {
	this.conflictMutex.Lock()
	defer this.conflictMutex.Unlock()
	this.currentConflictingAuthors.Add(author)
}
