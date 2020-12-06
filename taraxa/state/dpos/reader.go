package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Reader struct {
	cfg             *Config
	blk_n           types.BlockNum
	storage_factory func(types.BlockNum) StorageReader
	storage         *StorageReaderWrapper
	storage_past    *StorageReaderWrapper
}

func (self *Reader) Init(cfg *Config, blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) *Reader {
	self.cfg = cfg
	self.blk_n = blk_n
	self.storage_factory = storage_factory
	return self
}

func (self *Reader) get_storage() *StorageReaderWrapper {
	if self.storage == nil {
		self.storage = new(StorageReaderWrapper).Init(self.storage_factory(self.blk_n))
	}
	return self.storage
}

func (self *Reader) get_storage_past() *StorageReaderWrapper {
	if self.storage_past == nil {
		var blk_n uint64
		if self.cfg.DepositDelay < self.blk_n {
			blk_n = self.blk_n - self.cfg.DepositDelay
		}
		self.storage_past = new(StorageReaderWrapper).Init(self.storage_factory(blk_n))
	}
	return self.storage_past
}

func (self Reader) EligibleAddressCount() (ret uint64) {
	self.get_storage_past().Get(stor_k_1(field_eligible_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.GetStakingBalance(address).Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = bigutil.Big0
	self.get_storage_past().Get(stor_k_1(field_staking_balances, addr[:]), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

type Query struct {
	WithEligibleCount bool
	AccountQueries    map[common.Address]AccountQuery
}
type AccountQuery struct {
	WithStakingBalance                      bool
	DepositInfoCorrespondsToStakingBalances bool
	WithOutboundDeposits                    bool
	OutboundDepositsAddrsOnly               bool
	WithInboundDeposits                     bool
	InboundDepositsAddrsOnly                bool
}
type QueryResult struct {
	EligibleCount       uint64
	AccountQueryResults map[common.Address]*AccountQueryResult
}
type AccountQueryResult struct {
	StakingBalance   *big.Int
	IsEligible       bool
	OutboundDeposits map[common.Address]*DepositValue
	InboundDeposits  map[common.Address]*DepositValue
}

func (self Reader) Query(q *Query) (ret QueryResult) {
	if q.WithEligibleCount {
		ret.EligibleCount = self.EligibleAddressCount()
	}
	ret.AccountQueryResults = make(map[common.Address]*AccountQueryResult)
	for addr, q := range q.AccountQueries {
		res := new(AccountQueryResult)
		ret.AccountQueryResults[addr] = res
		if q.WithStakingBalance {
			res.StakingBalance = self.GetStakingBalance(&addr)
			res.IsEligible = self.cfg.EligibilityBalanceThreshold.Cmp(res.StakingBalance) <= 0
		}
		res.OutboundDeposits = make(map[common.Address]*DepositValue)
		res.InboundDeposits = make(map[common.Address]*DepositValue)
		for i := 0; i < 2; i++ {
			with, addrs_only, res_map := q.WithOutboundDeposits, q.OutboundDepositsAddrsOnly, res.OutboundDeposits
			list_kind := field_addrs_out
			if i%2 == 1 {
				with, addrs_only, res_map = q.WithInboundDeposits, q.InboundDepositsAddrsOnly, res.InboundDeposits
				list_kind = field_addrs_in
			}
			if !with {
				continue
			}
			var storage *StorageReaderWrapper
			if q.DepositInfoCorrespondsToStakingBalances {
				storage = self.get_storage_past()
			} else {
				storage = self.get_storage()
			}
			storage.ListForEach(bin.Concat2(list_kind, addr[:]), func(addr_other_raw []byte) {
				addr_other := common.BytesToAddress(addr_other_raw)
				var val *DepositValue
				if !addrs_only {
					addr1, addr2 := &addr, &addr_other
					if i%2 == 1 {
						addr1, addr2 = addr2, addr1
					}
					storage.Get(stor_k_1(field_deposits, addr1[:], addr2[:]), func(bytes []byte) {
						var deposit Deposit
						rlp.MustDecodeBytes(bytes, &deposit)
						val = &deposit.DepositValue
					})
				}
				res_map[addr_other] = val
			})
		}
	}
	return
}
