package pub_sub

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// TODO !!!
type nodeStatus = uint32

type terminator byte

const (
	pending nodeStatus = 0
	linked
	read
)

var genesisNode = unsafe.Pointer(new(byte))
var terminalNode = unsafe.Pointer(new(byte))

type node struct {
	prev        unsafe.Pointer
	value       interface{}
	hasBeenRead nodeStatus
}

type PubSub struct {
	head unsafe.Pointer
}

func NewPubSub() PubSub {
	return PubSub{head: genesis}
}

func (this *PubSub) Write(value interface{}) bool {
	if value == nil {
		atomic.CompareAndSwapPointer(this.head, )
	}
	newNode := &node{value: value}
	prev := atomic.SwapPointer(&this.head, unsafe.Pointer(newNode))
	if (*node)(prev).value == nil {
		atomic.StorePointer(&this.head, prev)
		return false
	}
	atomic.StorePointer(&newNode.prev, prev)
	if value == nil {

	}
	return true
}

func (this *PubSub) RequestTermination() bool {
	return this.Write(nil)
}

func (this *PubSub) NewReader() *Reader {
	return &Reader{headPtr: &this.head}
}

type Reader struct {
	headPtr           *unsafe.Pointer
	prevHead          *node
	toRead            *node
	currentHead       *node
	hasSeenTerminator bool
}

func (this *Reader) Read(unreadOnly bool) (val interface{}) {
	for val == nil {
		if this.toRead == this.prevHead {
			if this.hasSeenTerminator {
				return nil
			}
			this.prevHead = this.currentHead
			for {
				if this.toRead = (*node)(atomic.LoadPointer(this.headPtr)); this.toRead != this.currentHead {
					this.currentHead = this.toRead
					break
				}
				runtime.Gosched()
			}
		}
		if val = this.toRead.value; val == nil {
			this.hasSeenTerminator = true
		} else if !atomic.CompareAndSwapUint32(&this.toRead.hasBeenRead, 0, 1) && unreadOnly {
			val = nil
		}
		this.toRead = (*node)(this.toRead.prev)
	}
	return
}
