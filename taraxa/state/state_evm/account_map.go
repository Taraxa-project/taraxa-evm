package state_evm

import (
	"math"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
)

type AccountMap struct {
	num_entries                      uint64
	buckets                          []AccountMapBucket
	log2_buckets_count               uint32
	bucket_overflow_desired_capacity uint64
	bucket_pos_last                  uintptr
	hasher_seed                      uintptr
}
type AccountMapBucket = struct {
	entries          [AccountMapBucketBaseSize]*Account
	entries_overflow Accounts
}
type AccountMapEntryHeader = struct {
	AccountMapKey
	pos_of_bucket uint32
	pos_in_bucket uint32
}
type AccountMapKey = struct {
	hash_top uint32
	addr     common.Address
}
type AccountMapOptions = struct {
	NumBuckets                    uint64
	BucketOverflowDesiredCapacity uint64
}

const AccountMapBucketBaseSize = 3

func (self *AccountMap) Init(opts AccountMapOptions) *AccountMap {
	asserts.Holds(opts.NumBuckets > 0)
	asserts.Holds(opts.BucketOverflowDesiredCapacity > 0)
	self.hasher_seed = rand_uintptr() | 1
	self.log2_buckets_count = uint32(math.Ceil(math.Log2(float64(opts.NumBuckets))))
	self.buckets = make([]AccountMapBucket, 1<<self.log2_buckets_count)
	self.bucket_pos_last = uintptr(len(self.buckets) - 1)
	self.bucket_overflow_desired_capacity = opts.BucketOverflowDesiredCapacity
	return self
}

func (self *AccountMap) GetOrNew(addr *common.Address) (ret *Account, was_present bool) {
	defer func() {
		if !was_present {
			self.num_entries++
		}
	}()
	hash := hash_addr(addr, self.hasher_seed)
	pos_of_bucket := uint32(hash & self.bucket_pos_last)
	hash_top := uint32(hash >> self.log2_buckets_count)
	bucket, pos := &self.buckets[pos_of_bucket], uint32(0)
	if bucket.entries[1] != nil {
		key := AccountMapKey{hash_top, *addr}
		for max_size := uint32(AccountMapBucketBaseSize + len(bucket.entries_overflow)); pos < max_size; pos++ {
			if pos < AccountMapBucketBaseSize {
				if ret = bucket.entries[pos]; ret == nil {
					break
				}
			} else {
				ret = bucket.entries_overflow[pos-AccountMapBucketBaseSize]
			}
			if was_present = ret.AccountMapKey == key; was_present {
				return
			}
		}
		ret = &Account{AccountMapEntryHeader: AccountMapEntryHeader{key, pos_of_bucket, pos}}
		if pos < AccountMapBucketBaseSize {
			bucket.entries[pos] = ret
		} else {
			if bucket.entries_overflow == nil {
				bucket.entries_overflow = make(Accounts, 0, self.bucket_overflow_desired_capacity)
			}
			bucket.entries_overflow = append(bucket.entries_overflow, ret)
		}
		return
	}
	if ret = bucket.entries[0]; ret != nil {
		if was_present = ret.addr == *addr; was_present {
			return
		}
		pos++
	}
	ret = &Account{AccountMapEntryHeader: AccountMapEntryHeader{AccountMapKey{hash_top, *addr}, pos_of_bucket, pos}}
	bucket.entries[pos] = ret
	return
}

func (self *AccountMap) Delete(acc *Account) (was_present bool) {
	if acc.pos_in_bucket == math.MaxUint32 {
		return
	}
	self.num_entries--
	bucket, pos := &self.buckets[acc.pos_of_bucket], acc.pos_in_bucket
	acc.pos_in_bucket = math.MaxUint32
	var last_acc *Account
	if last_overflow_pos := len(bucket.entries_overflow) - 1; last_overflow_pos != -1 {
		last_acc = bucket.entries_overflow[last_overflow_pos]
		switch last_overflow_pos {
		case int(self.bucket_overflow_desired_capacity):
			tmp := make(Accounts, self.bucket_overflow_desired_capacity)
			copy(tmp, bucket.entries_overflow[:last_overflow_pos])
			bucket.entries_overflow = tmp
		case 0:
			bucket.entries_overflow = nil
		default:
			bucket.entries_overflow[last_overflow_pos] = nil
			bucket.entries_overflow = bucket.entries_overflow[:last_overflow_pos]
		}
		if last_overflow_pos == int(pos-AccountMapBucketBaseSize) {
			return true
		}
	} else {
		for i := pos + 1; ; i++ {
			if i < AccountMapBucketBaseSize {
				if acc := bucket.entries[i]; acc != nil {
					last_acc = acc
					continue
				}
			}
			last_pos := i - 1
			bucket.entries[last_pos] = nil
			if last_pos == pos {
				return true
			}
			break
		}
	}
	if last_acc.pos_in_bucket = pos; pos < AccountMapBucketBaseSize {
		bucket.entries[pos] = last_acc
	} else {
		bucket.entries_overflow[pos-AccountMapBucketBaseSize] = last_acc
	}
	return true
}
