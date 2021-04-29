package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000ff"))

var field_staking_balances = []byte{0}
var field_deposits = []byte{1}
var field_eligible_count = []byte{2}
var field_withdrawals_by_block = []byte{3}
var field_addrs_in = []byte{4}
var field_addrs_out = []byte{5}
var field_eligible_vote_count = []byte{6}
var field_amount_delegated = []byte{7}

var ErrTransferAmountIsZero = util.ErrorString("transfer amount is zero")
var ErrWithdrawalExceedsDeposit = util.ErrorString("withdrawal exceeds prior deposit value")
var ErrInsufficientBalanceForDeposits = util.ErrorString("insufficient balance for the deposits")
var ErrCallIsNotToplevel = util.ErrorString("only top-level calls are allowed")
var ErrNoTransfers = util.ErrorString("no transfers")
var ErrCallValueNonzero = util.ErrorString("call value must be zero")
var ErrDuplicateBeneficiary = util.ErrorString("duplicate beneficiary")

type Contract struct {
	cfg                           Config
	storage                       StorageWrapper
	staking_balances              Addr2Balance
	deposits                      map[common.Hash]*Deposit
	amount_delegated              *big.Int
	amount_delegated_initialized  bool
	eligible_count                uint64
	eligible_vote_count           uint64
	eligible_count_initialized    bool
	eligible_count_dirty          bool
	curr_withdrawals              Addr2Addr2Balance
}
type Addr2Balance = map[common.Address]*big.Int
type Addr2Addr2Balance = map[common.Address]Addr2Balance
type Transfer = struct {
	Value    *big.Int
	Negative bool
}
type BeneficiaryAndTransfer struct {
	Beneficiary common.Address
	Transfer    Transfer
}
type Transfers = []BeneficiaryAndTransfer

type Deposit struct {
	ValueNet               *big.Int
	ValuePendingWithdrawal *big.Int
	AddrsInPos             uint64
	AddrsOutPos            uint64
}

// Returns the number of times y can be subtracted from x
// and remain positive
func EligibleVotes(x, y *big.Int) (ret uint64) {
	ret = 0
	tot := bigutil.Big0
	tot = bigutil.Add(tot, x)
	for tot.Cmp(y) >= 0 {
		tot = bigutil.USub(tot, y)
		ret++
	}
	return
}


func (self *Deposit) Init() *Deposit {
	self.ValueNet, self.ValuePendingWithdrawal = bigutil.Big0, bigutil.Big0
	return self
}

func (self *Deposit) Total() *big.Int {
	return bigutil.Add(self.ValueNet, self.ValuePendingWithdrawal)
}

func (self *Deposit) IsZero() bool {
	return bigutil.IsZero(self.ValueNet) && bigutil.IsZero(self.ValuePendingWithdrawal)
}

func (self *Contract) init(cfg Config, storage Storage) *Contract {
	self.cfg = cfg
	self.storage.Init(storage)
	return self
}

func (self *Contract) ApplyGenesis() error {
	for _, entry := range self.cfg.GenesisState {
		transfers := make([]BeneficiaryAndTransfer, len(entry.Transfers))
		for i, v := range entry.Transfers {
			transfers[i] = BeneficiaryAndTransfer{
				Beneficiary: v.Beneficiary,
				Transfer:    Transfer{Value: v.Value},
			}
		}
		if err := self.run(entry.Benefactor, transfers); err != nil {
			return err
		}
	}
	self.storage.IncrementNonce(contract_address)
	self.Commit(0)
	return nil
}

func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	return uint64(len(ctx.Input)) * 20 // TODO
}

func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if ctx.Value.Sign() != 0 {
		return nil, ErrCallValueNonzero
	}
	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}
	var transfers Transfers
	if err := rlp.DecodeBytes(ctx.Input, &transfers); err != nil {
		return nil, err
	}
	return nil, self.run(*ctx.CallerAccount.Address(), transfers)
}

func (self *Contract) run(benefactor common.Address, transfers Transfers) (err error) {
	if len(transfers) == 0 {
		return ErrNoTransfers
	}
	expenditure_total := bigutil.Big0
	unique_beneficiaries := make(map[common.Address]bool, len(transfers))
	for _, t := range transfers {
		if unique_beneficiaries[t.Beneficiary] {
			return ErrDuplicateBeneficiary
		}
		unique_beneficiaries[t.Beneficiary] = true
		if t.Transfer.Value.Sign() == 0 {
			return ErrTransferAmountIsZero
		}
		if !t.Transfer.Negative {
			expenditure_total = bigutil.Add(expenditure_total, t.Transfer.Value)
		} else {
			deposit, _ := self.deposits_get(benefactor[:], t.Beneficiary[:])
			if deposit == nil || deposit.ValueNet.Cmp(t.Transfer.Value) < 0 {
				return ErrWithdrawalExceedsDeposit
			}
		}
	}
	if !self.storage.SubBalance(&benefactor, expenditure_total) {
		return ErrInsufficientBalanceForDeposits
	}
	for _, t := range transfers {
		beneficiary, transfer := t.Beneficiary, t.Transfer
		deposit, deposit_k := self.deposits_get(benefactor[:], beneficiary[:])
		if deposit == nil {
			deposit = new(Deposit).Init()
		}
		op := bigutil.Add
		if transfer.Negative {
			op = bigutil.USub
			if self.curr_withdrawals == nil {
				self.curr_withdrawals = make(Addr2Addr2Balance)
			}
			benefactor_withdrawals := self.curr_withdrawals[benefactor]
			if benefactor_withdrawals == nil {
				benefactor_withdrawals = make(Addr2Balance)
				self.curr_withdrawals[benefactor] = benefactor_withdrawals
			}
			benefactor_withdrawals[beneficiary] = bigutil.Add(benefactor_withdrawals[beneficiary], transfer.Value)
			deposit.ValuePendingWithdrawal = bigutil.Add(deposit.ValuePendingWithdrawal, transfer.Value)
		} else {
			self.upd_staking_balance(beneficiary, transfer.Value, false)
			if deposit.IsZero() {
				deposit.AddrsOutPos = self.storage.ListAppend(
					bin.Concat2(field_addrs_out, benefactor[:]),
					common.CopyBytes(beneficiary[:]))
				deposit.AddrsInPos = self.storage.ListAppend(
					bin.Concat2(field_addrs_in, beneficiary[:]),
					common.CopyBytes(benefactor[:]))
			}
		}
		deposit.ValueNet = op(deposit.ValueNet, transfer.Value)
		self.deposits_put(&deposit_k, deposit)
	}
	return
}

func (self *Contract) Commit(blk_n types.BlockNum) {
	defer self.storage.ClearCache()
	var moneyback_withdrawals Addr2Addr2Balance
	if self.cfg.WithdrawalDelay == 0 {
		moneyback_withdrawals = self.curr_withdrawals
	} else {
		if len(self.curr_withdrawals) != 0 {
			self.storage.Put(
				stor_k_1(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n)),
				rlp.MustEncodeToBytes(self.curr_withdrawals))
		}
		if self.cfg.WithdrawalDelay < blk_n {
			k := stor_k_1(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n-self.cfg.WithdrawalDelay))
			self.storage.Get(k, func(bytes []byte) {
				moneyback_withdrawals = make(Addr2Addr2Balance)
				rlp.MustDecodeBytes(bytes, &moneyback_withdrawals)
			})
			self.storage.Put(k, nil)
		}
	}
	for benefactor, withdrawal_per_beneficiary := range moneyback_withdrawals {
		val_total := bigutil.Big0
		for _, val := range withdrawal_per_beneficiary {
			val_total = bigutil.Add(val_total, val)
		}
		self.storage.AddBalance(&benefactor, val_total)
	}
	var withdrawals_to_apply Addr2Addr2Balance
	if self.cfg.DepositDelay == 0 {
		withdrawals_to_apply = moneyback_withdrawals
	} else if delay_diff := self.cfg.WithdrawalDelay - self.cfg.DepositDelay; delay_diff == 0 {
		withdrawals_to_apply = self.curr_withdrawals
	} else if delay_diff < blk_n {
		self.storage.Get(
			stor_k_1(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n-delay_diff)),
			func(bytes []byte) {
				withdrawals_to_apply = make(Addr2Addr2Balance)
				rlp.MustDecodeBytes(bytes, &withdrawals_to_apply)
			})
	}
	for benefactor, withdrawal_per_beneficiary := range withdrawals_to_apply {
		for beneficiary, val := range withdrawal_per_beneficiary {
			self.upd_staking_balance(beneficiary, val, true)
			deposit, deposit_k := self.deposits_get(benefactor[:], beneficiary[:])
			deposit.ValuePendingWithdrawal = bigutil.USub(deposit.ValuePendingWithdrawal, val)
			if !deposit.IsZero() {
				self.deposits_put(&deposit_k, deposit)
				continue
			}
			for i := 0; i < 2; i++ {
				list_kind, list_owner, pos := field_addrs_out, benefactor[:], deposit.AddrsOutPos
				if i%2 == 1 {
					list_kind, list_owner, pos = field_addrs_in, beneficiary[:], deposit.AddrsInPos
				}
				moved_addr := self.storage.ListRemove(bin.Concat2(list_kind, list_owner), pos)
				if moved_addr == nil {
					continue
				}
				addr1, addr2 := list_owner, moved_addr
				if i%2 == 1 {
					addr1, addr2 = addr2, addr1
				}
				deposit, deposit_k := self.deposits_get(addr1, addr2)
				if i%2 == 0 {
					deposit.AddrsOutPos = pos
				} else {
					deposit.AddrsInPos = pos
				}
				self.deposits_put(&deposit_k, deposit)
			}
			self.deposits_put(&deposit_k, nil)
		}
	}
	if self.eligible_count_dirty {
		self.eligible_count_dirty = false
		self.storage.Put(stor_k_1(field_eligible_count), bin.ENC_b_endian_compact_64_1(self.eligible_count))
		self.storage.Put(stor_k_1(field_eligible_vote_count), bin.ENC_b_endian_compact_64_1(self.eligible_vote_count))	
	}
	self.storage.Put(stor_k_1(field_amount_delegated), self.amount_delegated.Bytes())
	self.staking_balances, self.deposits, self.curr_withdrawals = nil, nil, nil
}

func (self *Contract) upd_staking_balance(beneficiary common.Address, delta *big.Int, negative bool) {
	if !self.amount_delegated_initialized {
		self.amount_delegated_initialized = true
		self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
			self.amount_delegated = bigutil.FromBytes(bytes)
		})
	}
	if self.staking_balances == nil {
		self.staking_balances = make(Addr2Balance)
	}
	beneficiary_bal := self.staking_balances[beneficiary]
	if beneficiary_bal == nil {
		beneficiary_bal = bigutil.Big0
		self.storage.Get(stor_k_1(field_staking_balances, beneficiary[:]), func(bytes []byte) {
			beneficiary_bal = bigutil.FromBytes(bytes)
		})
	}
	prev_beneficiary_bal :=  bigutil.Big0
	prev_beneficiary_bal = bigutil.Add(prev_beneficiary_bal, beneficiary_bal)
	
	was_eligible := beneficiary_bal.Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
	if negative {
		beneficiary_bal = bigutil.USub(beneficiary_bal, delta)
	} else {
		beneficiary_bal = bigutil.Add(beneficiary_bal, delta)
	}
	self.staking_balances[beneficiary] = beneficiary_bal
	self.storage.Put(stor_k_1(field_staking_balances, beneficiary[:]), beneficiary_bal.Bytes())
	eligible_now := beneficiary_bal.Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
	eligible_count_change := 0
	new_vote_count := EligibleVotes(beneficiary_bal, self.cfg.EligibilityBalanceThreshold)
	prev_vote_count := EligibleVotes(prev_beneficiary_bal, self.cfg.EligibilityBalanceThreshold)
	if was_eligible && !eligible_now {
		eligible_count_change = -1
		self.amount_delegated = bigutil.USub(self.amount_delegated, prev_beneficiary_bal)
	}
	if !was_eligible && eligible_now {
		eligible_count_change = 1
		self.amount_delegated = bigutil.Add(self.amount_delegated, beneficiary_bal)
	}
	if eligible_count_change == 0 {
		if negative {
			self.amount_delegated = bigutil.USub(self.amount_delegated, delta)
		} else {
			self.amount_delegated = bigutil.Add(self.amount_delegated, delta)
		}
	}
	if eligible_count_change == 0 && new_vote_count == prev_vote_count {
		return
	}
	self.eligible_count_dirty = true
	if !self.eligible_count_initialized {
		self.eligible_count_initialized = true
		self.storage.Get(stor_k_1(field_eligible_count), func(bytes []byte) {
			self.eligible_count = bin.DEC_b_endian_compact_64(bytes)
		})
		self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
			self.eligible_vote_count = bin.DEC_b_endian_compact_64(bytes)
		})
	}
	if eligible_count_change == 1 {
		self.eligible_count++
	} else if eligible_count_change == -1 {
		self.eligible_count--
	}
	self.eligible_vote_count += new_vote_count
	self.eligible_vote_count -= prev_vote_count
}

func (self *Contract) deposits_get(benefactor_addr, beneficiary_addr []byte) (deposit *Deposit, key common.Hash) {
	key = stor_k_2(field_deposits, benefactor_addr, beneficiary_addr)
	if val, ok := self.deposits[key]; ok {
		deposit = val
		return
	}
	self.storage.Get(&key, func(bytes []byte) {
		deposit = new(Deposit)
		rlp.MustDecodeBytes(bytes, deposit)
	})
	if self.deposits == nil {
		self.deposits = make(map[common.Hash]*Deposit)
	}
	self.deposits[key] = deposit
	return
}

func (self *Contract) deposits_put(key *common.Hash, deposit *Deposit) {
	if deposit != nil {
		self.storage.Put(key, rlp.MustEncodeToBytes(deposit))
	} else {
		self.storage.Put(key, nil)
	}
	if self.deposits == nil {
		self.deposits = make(map[common.Hash]*Deposit)
	}
	self.deposits[*key] = deposit
}
