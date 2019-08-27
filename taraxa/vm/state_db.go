package vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"sync"
)

type StateReader interface {
	GetBalance(common.Address) *big.Int
	HasBalance(address common.Address, amount *big.Int) bool
	GetNonce(common.Address) uint64
	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	GetCodeSize(common.Address) int
	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	HasSuicided(common.Address) bool
	Exist(common.Address) bool
	Empty(common.Address) bool
}

type StateWriter interface {
	CreateAccount(common.Address)
	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	AddNonce(common.Address, uint64)
	SetCode(common.Address, []byte)
	SetState(common.Address, common.Hash, common.Hash)
	Suicide(common.Address) bool
}

type State interface {
	StateReader
	StateWriter
}

type TransactionState interface {
	State
	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64
	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)
	RevertToSnapshot(int)
	Snapshot() int
}

type shard struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

type ShardedMap struct {
	shardCount uint64
	shards     []*shard
}

func NewShardedMap(shardCount uint64) *ShardedMap {
	this := &ShardedMap{shardCount, make([]*shard, shardCount)}
	for i := uint64(0); i < shardCount; i++ {
		this.shards[i] = &shard{data: make(map[common.Address]interface{})}
	}
	return this
}

func (this *ShardedMap) getShard(k string) *shard {
	return this.shards[util.CRC64([]byte(k))%this.shardCount]
}

func (this *ShardedMap) Get(k string) (val interface{}, ok bool) {
	shard := this.getShard(k)
	defer concurrent.LockUnlock(shard.mu.RLocker())()
	val, ok = shard.data[k]
	return
}

func (this *ShardedMap) Put(k string, v interface{}) (prev interface{}, prevHasBeen bool) {
	shard := this.getShard(k)
	defer concurrent.LockUnlock(&shard.mu)()
	prev, prevHasBeen = shard.data[k]
	shard.data[k] = v
	return
}

const AccountFieldLength = 1 + common.HashLength
type AccountField [AccountFieldLength]byte
type StateField = [common.AddressLength + AccountFieldLength]byte

const balance = AccountField{0, common.Hash{}}
const (
	balance = iota
	nonce
	codeHash
	storageRoot
	storage
	code
	AccountFields_count
)

type ConcurrentStateCache struct {
	rawReader func([]byte) ([]byte, error)
	state     *ShardedMap
}

func (this *ConcurrentStateCache) GetBalance(addr common.Address) *big.Int {
	accountKey := string(addr[:])
	balanceKey := append(addr, balance)
	this.state.Get(addr.String())
}

func (this *ConcurrentStateCache) HasBalance(address common.Address, amount *big.Int) bool {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetNonce(common.Address) uint64 {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetCodeHash(common.Address) common.Hash {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetCode(common.Address) []byte {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetCodeSize(common.Address) int {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetCommittedState(common.Address, common.Hash) common.Hash {
	panic("implement me")
}

func (this *ConcurrentStateCache) GetState(common.Address, common.Hash) common.Hash {
	panic("implement me")
}

func (this *ConcurrentStateCache) HasSuicided(common.Address) bool {
	panic("implement me")
}

func (this *ConcurrentStateCache) Exist(common.Address) bool {
	panic("implement me")
}

func (this *ConcurrentStateCache) Empty(common.Address) bool {
	panic("implement me")
}
