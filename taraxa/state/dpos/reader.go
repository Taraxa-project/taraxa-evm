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
	cfg     *Config
	storage *StorageReaderWrapper
}

func (self *Reader) Init(cfg *Config, blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) *Reader {
	self.cfg = cfg
	var blk_n_actual types.BlockNum
	if self.cfg.DepositDelay < blk_n {
		blk_n_actual = blk_n - self.cfg.DepositDelay
	}
	self.storage = new(StorageReaderWrapper).Init(storage_factory(blk_n_actual))
	return self
}

func (self Reader) EligibleAddressCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) EligibleVoteCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
	return vote_count(self.GetStakingBalance(addr), self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
}

func (self Reader) TotalAmountDelegated() (ret *big.Int) {
	ret = bigutil.Big0
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.cfg.EligibilityBalanceThreshold.Cmp(self.GetStakingBalance(address)) <= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = bigutil.Big0
	self.storage.Get(stor_k_1(field_staking_balances, addr[:]), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

type Query struct {
	WithEligibleCount bool
	AccountQueries    map[common.Address]AccountQuery
}
type AccountQuery struct {
	WithStakingBalance        bool
	WithOutboundDeposits      bool
	OutboundDepositsAddrsOnly bool
	WithInboundDeposits       bool
	InboundDepositsAddrsOnly  bool
}
type QueryResult struct {
	EligibleCount  uint64
	AccountResults map[common.Address]*AccountQueryResult
}
type AccountQueryResult struct {
	StakingBalance   *big.Int
	IsEligible       bool
	OutboundDeposits Addr2Balance
	InboundDeposits  Addr2Balance
}

func (self Reader) Query(q *Query) (ret QueryResult) {
	if q.WithEligibleCount {
		ret.EligibleCount = self.EligibleAddressCount()
	}
	ret.AccountResults = make(map[common.Address]*AccountQueryResult)
	for addr, q := range q.AccountQueries {
		res := &AccountQueryResult{
			OutboundDeposits: make(Addr2Balance),
			InboundDeposits:  make(Addr2Balance),
		}
		ret.AccountResults[addr] = res
		if q.WithStakingBalance {
			res.StakingBalance = self.GetStakingBalance(&addr)
			res.IsEligible = self.cfg.EligibilityBalanceThreshold.Cmp(res.StakingBalance) <= 0
		}
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
			self.storage.ListForEach(bin.Concat2(list_kind, addr[:]), func(addr_other_raw []byte) {
				addr_other := common.BytesToAddress(addr_other_raw)
				var deposit_v *big.Int
				if !addrs_only {
					addr1, addr2 := &addr, &addr_other
					if i%2 == 1 {
						addr1, addr2 = addr2, addr1
					}
					self.storage.Get(stor_k_1(field_deposits, addr1[:], addr2[:]), func(bytes []byte) {
						var deposit Deposit
						rlp.MustDecodeBytes(bytes, &deposit)
						deposit_v = deposit.Total()
					})
				}
				res_map[addr_other] = deposit_v
			})
		}
	}
	return
}
