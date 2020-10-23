package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000ff"))

var field_staking_balances = []byte{0}
var field_deposits = []byte{1}
var field_eligible_count = []byte{2}
var field_withdrawals_by_block = []byte{3}

var ErrTransferAmountIsZero = util.ErrorString("transfer amount is zero")
var ErrWithdrawalExceedsDeposit = util.ErrorString("withdrawal exceeds prior deposit value")
var ErrInsufficientBalanceForDeposits = util.ErrorString("insufficient balance for the deposits")
var ErrCallIsNotToplevel = util.ErrorString("only top-level calls are allowed")
var ErrNoTransfers = util.ErrorString("no transfers")
var ErrCallValueNonzero = util.ErrorString("call value must be zero")

type Contract struct {
	cfg                        Config
	storage                    Storage
	staking_balances           BalanceMap
	deposits                   DelegatedBalanceMap
	eligible_count             uint64
	eligible_count_initialized bool
	eligible_count_dirty       bool
	curr_withdrawals           DelegatedBalanceMap
}
type benefactor_t = common.Address
type beneficiary_t = common.Address
type BalanceMap = map[beneficiary_t]*big.Int
type DelegatedBalanceMap = map[benefactor_t]BalanceMap
type Transfer = struct {
	Value    *big.Int
	Negative bool
}
type Transfers = map[beneficiary_t]Transfer
type Storage interface {
	SubBalance(*common.Address, *big.Int) bool
	AddBalance(*common.Address, *big.Int)
	Put(*common.Address, *common.Hash, []byte)
	Get(*common.Address, *common.Hash, func([]byte))
	IncrementNonce(address *common.Address)
}

func (self *Contract) init(cfg Config, storage Storage) *Contract {
	self.cfg = cfg
	self.storage = storage
	return self
}

func (self *Contract) ApplyGenesis() error {
	for benefactor, benefactor_deposits := range self.cfg.GenesisState {
		transfers := make(map[beneficiary_t]Transfer, len(benefactor_deposits))
		for k, v := range benefactor_deposits {
			transfers[k] = Transfer{Value: v}
		}
		if err := self.run(benefactor, transfers); err != nil {
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
	if self.deposits == nil {
		self.deposits = make(DelegatedBalanceMap)
	}
	benefactor_deposits := self.deposits[benefactor]
	if benefactor_deposits == nil {
		benefactor_deposits = make(BalanceMap)
		self.deposits[benefactor] = benefactor_deposits
	}
	expenditure_total := bigutil.Big0
	for beneficiary, transfer := range transfers {
		if transfer.Value.Sign() == 0 {
			return ErrTransferAmountIsZero
		}
		deposit_v := benefactor_deposits[beneficiary]
		if deposit_v == nil {
			deposit_v = bigutil.Big0
			self.storage.Get(contract_address, stor_k(field_deposits, benefactor[:], beneficiary[:]), func(bytes []byte) {
				deposit_v = bigutil.FromBytes(bytes)
			})
			benefactor_deposits[beneficiary] = deposit_v
		}
		if !transfer.Negative {
			expenditure_total = new(big.Int).Add(expenditure_total, transfer.Value)
		} else if deposit_v.Cmp(transfer.Value) < 0 {
			return ErrWithdrawalExceedsDeposit
		}
	}
	if !self.storage.SubBalance(&benefactor, expenditure_total) {
		return ErrInsufficientBalanceForDeposits
	}
	for beneficiary, transfer := range transfers {
		op := bigutil.Add
		if transfer.Negative {
			op = bigutil.USub
			if self.curr_withdrawals == nil {
				self.curr_withdrawals = make(DelegatedBalanceMap)
			}
			benefactor_withdrawals := self.curr_withdrawals[benefactor]
			if benefactor_withdrawals == nil {
				benefactor_withdrawals = make(BalanceMap)
				self.curr_withdrawals[benefactor] = benefactor_withdrawals
			}
			benefactor_withdrawals[beneficiary] = bigutil.Add(benefactor_withdrawals[beneficiary], transfer.Value)
		} else {
			self.upd_staking_balance(beneficiary, transfer.Value, false)
		}
		deposit_v := op(benefactor_deposits[beneficiary], transfer.Value)
		benefactor_deposits[beneficiary] = deposit_v
		self.storage.Put(contract_address, stor_k(field_deposits, benefactor[:], beneficiary[:]), deposit_v.Bytes())
	}
	return
}

func (self *Contract) Commit(blk_n types.BlockNum) {
	var moneyback_withdrawals DelegatedBalanceMap
	if self.cfg.WithdrawalDelay == 0 {
		moneyback_withdrawals = self.curr_withdrawals
	} else {
		if len(self.curr_withdrawals) != 0 {
			self.storage.Put(
				contract_address,
				stor_k(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n)),
				rlp.MustEncodeToBytes(self.curr_withdrawals))
		}
		if self.cfg.WithdrawalDelay < blk_n {
			self.storage.Get(
				contract_address,
				stor_k(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n-self.cfg.WithdrawalDelay)),
				func(bytes []byte) {
					moneyback_withdrawals = make(DelegatedBalanceMap)
					rlp.DecodeBytes(bytes, &moneyback_withdrawals)
				})
		}
	}
	for benefactor, withdrawal_per_beneficiary := range moneyback_withdrawals {
		val_total := bigutil.Big0
		for _, val := range withdrawal_per_beneficiary {
			val_total = bigutil.Add(val_total, val)
		}
		self.storage.AddBalance(&benefactor, val_total)
	}
	var withdrawals_to_apply DelegatedBalanceMap
	if self.cfg.DepositDelay == 0 {
		withdrawals_to_apply = moneyback_withdrawals
	} else if delay_diff := self.cfg.WithdrawalDelay - self.cfg.DepositDelay; delay_diff == 0 {
		withdrawals_to_apply = self.curr_withdrawals
	} else if delay_diff < blk_n {
		self.storage.Get(
			contract_address,
			stor_k(field_withdrawals_by_block, bin.ENC_b_endian_compact_64_1(blk_n-delay_diff)),
			func(bytes []byte) {
				withdrawals_to_apply = make(DelegatedBalanceMap)
				rlp.DecodeBytes(bytes, &withdrawals_to_apply)
			})
	}
	for _, withdrawal_per_beneficiary := range withdrawals_to_apply {
		for beneficiary, val := range withdrawal_per_beneficiary {
			self.upd_staking_balance(beneficiary, val, true)
		}
	}
	if self.eligible_count_dirty {
		self.eligible_count_dirty = false
		self.storage.Put(
			contract_address,
			stor_k(field_eligible_count),
			bin.ENC_b_endian_compact_64_1(self.eligible_count))
	}
	self.staking_balances, self.deposits, self.curr_withdrawals = nil, nil, nil
}

func (self *Contract) upd_staking_balance(beneficiary common.Address, delta *big.Int, negative bool) {
	if self.staking_balances == nil {
		self.staking_balances = make(BalanceMap)
	}
	beneficiary_bal := self.staking_balances[beneficiary]
	if beneficiary_bal == nil {
		beneficiary_bal = bigutil.Big0
		self.storage.Get(
			contract_address,
			stor_k(field_staking_balances, beneficiary[:]),
			func(bytes []byte) {
				beneficiary_bal = bigutil.FromBytes(bytes)
			})
	}
	was_eligible := beneficiary_bal.Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
	if negative {
		beneficiary_bal = bigutil.USub(beneficiary_bal, delta)
	} else {
		beneficiary_bal = bigutil.Add(beneficiary_bal, delta)
	}
	self.staking_balances[beneficiary] = beneficiary_bal
	self.storage.Put(
		contract_address,
		stor_k(field_staking_balances, beneficiary[:]),
		beneficiary_bal.Bytes())
	eligible_now := beneficiary_bal.Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
	eligible_count_change := 0
	if was_eligible && !eligible_now {
		eligible_count_change = -1
	}
	if !was_eligible && eligible_now {
		eligible_count_change = 1
	}
	if eligible_count_change == 0 {
		return
	}
	self.eligible_count_dirty = true
	if !self.eligible_count_initialized {
		self.eligible_count_initialized = true
		self.storage.Get(contract_address, stor_k(field_eligible_count), func(bytes []byte) {
			self.eligible_count = bin.DEC_b_endian_compact_64(bytes)
		})
	}
	if eligible_count_change == 1 {
		self.eligible_count++
	} else {
		self.eligible_count--
	}
}

func stor_k(parts ...[]byte) *common.Hash {
	return keccak256.Hash(parts...)
}
