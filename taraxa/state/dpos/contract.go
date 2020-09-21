package dpos

import (
	"errors"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/solidity_map"

	"github.com/Taraxa-project/taraxa-evm/common/math"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

var ContractAddress = common.HexToAddress("0x00000000000000000000000000000000000000ff")
var EligibleAddrSetAddress = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")

var field_balances = []byte{0}
var field_deposits = []byte{1}
var field_withdrawals = []byte{2}

type Contract struct {
	cfg                     Config
	storage                 Storage
	main_storage_helper     solidity_map.Map
	eligible_addr_hashes    map[beneficiary_t_hash]state_common.TaraxaBalance
	eligible_balances_dirty map[common.Address]state_common.TaraxaBalance
	balances                BalanceMap
	deposits                map[depositor_t]BalanceMap
	curr_withdrawals        []PendingWithdrawal
}
type depositor_t = common.Address
type beneficiary_t = common.Address
type beneficiary_t_hash = common.Hash
type Transfer = struct {
	Amount       state_common.TaraxaBalance
	IsWithdrawal bool
}
type BalanceMap = map[beneficiary_t]state_common.TaraxaBalance
type InboundTransfers = map[beneficiary_t]Transfer
type OutboundTransfers = map[depositor_t]InboundTransfers
type PendingWithdrawal = struct {
	Depositor   depositor_t
	Beneficiary beneficiary_t
	Amount      state_common.TaraxaBalance
}
type GenesisConfig struct {
	Transfers OutboundTransfers
}
type Storage interface {
	SubBalance(*common.Address, state_common.TaraxaBalance) bool
	Put(*common.Address, *common.Hash, []byte)
	Get(*common.Address, *common.Hash, func([]byte))
	ForEach(*common.Address, func(*common.Hash, []byte))
}

func (self *Contract) Init(cfg Config, storage Storage) *Contract {
	self.cfg = cfg
	self.storage = storage
	self.main_storage_helper.Init(solidity_map.Storage{
		Put: func(hash *common.Hash, bytes []byte) {
			self.storage.Put(&EligibleAddrSetAddress, hash, bytes)
		},
		Get: func(hash *common.Hash, cb func([]byte)) {
			self.storage.Get(&EligibleAddrSetAddress, hash, cb)
		},
	})
	self.eligible_addr_hashes = make(map[beneficiary_t_hash]state_common.TaraxaBalance)
	storage.ForEach(&EligibleAddrSetAddress, func(storage_key *common.Hash, val []byte) {
		self.eligible_addr_hashes[*storage_key] = bin.DEC_b_endian_compact_64(val)
	})
	return self
}

func (self *Contract) GenesisInit(cfg GenesisConfig) error {
	for depositor, transfers_in := range cfg.Transfers {
		if err := self.run(depositor, transfers_in); err != nil {
			return err
		}
	}
	self.Commit()
	return nil
}

func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	registry(&ContractAddress, self)
}

func (self *Contract) RequiredGas(ctx *vm.CallFrame, evm *vm.EVM) uint64 {
	return uint64(len(ctx.Input)) * 20
}

func (self *Contract) Run(ctx *vm.CallFrame, env *vm.ExecutionEnvironment) ([]byte, error) {
	if ctx.Value.Sign() != 0 {
		return nil, errors.New("call value must be zero")
	}
	if env.Depth != 1 {
		return nil, errors.New("only top-level calls are allowed")
	}
	var transfers InboundTransfers
	if err := rlp.DecodeBytes(ctx.Input, &transfers); err != nil {
		return nil, err
	}
	return nil, self.run(*ctx.CallerAccount.Address(), transfers)
}

func (self *Contract) run(depositor common.Address, transfers InboundTransfers) (err error) {
	var deposit_total state_common.TaraxaBalance
	for beneficiary, transfer := range transfers {
		if transfer.Amount == 0 {
			return errors.New("transfer amount is zero")
		}
		if transfer.IsWithdrawal {
			deposit, present := self.deposits[depositor][beneficiary]
			if !present {
				self.main_storage_helper.Get(
					func(bytes []byte) {
						deposit = bin.DEC_b_endian_compact_64(bytes)
					},
					field_deposits, depositor[:], beneficiary[:])
				self.deposits[depositor][beneficiary] = deposit
			}
			if deposit < transfer.Amount {
				return errors.New("withdrawal exceeds deposit")
			}
		} else if math.MaxUint64-deposit_total < transfer.Amount {
			return errors.New("total deposit value is impossibly large")
		} else {
			deposit_total += transfer.Amount
		}
	}
	if !self.storage.SubBalance(&depositor, deposit_total) {
		return errors.New("insufficient balance for the deposits")
	}
	for beneficiary, transfer := range transfers {
		if transfer.IsWithdrawal {
			self.deposits[depositor][beneficiary] -= transfer.Amount
			self.curr_withdrawals = append(self.curr_withdrawals, PendingWithdrawal{depositor, beneficiary, transfer.Amount})
		} else {
			self.deposits[depositor][beneficiary] += transfer.Amount
			bal_old, present := self.balances[beneficiary]
			if !present {
				self.main_storage_helper.Get(
					func(bytes []byte) {
						bal_old = bin.DEC_b_endian_compact_64(bytes)
					},
					field_balances, beneficiary[:])
			}
			bal_new := bal_old + transfer.Amount
			self.balances[beneficiary] = bal_new
			self.main_storage_helper.Put(bin.ENC_b_endian_compact_64_1(bal_new), field_balances, beneficiary[:])
			if bal_old < self.cfg.EligibilityBalanceThreshold && bal_new >= self.cfg.EligibilityBalanceThreshold {
				self.eligible_balances_dirty[beneficiary] = bal_new
			}
		}
		self.main_storage_helper.Put(bin.ENC_b_endian_compact_64_1(self.deposits[depositor][beneficiary]),
			field_deposits, depositor[:], beneficiary[:])
	}
	return
}

func (self *Contract) Commit() {

}

func (self *Contract) EligibleAddressCount() state_common.TaraxaBalance {
	return state_common.TaraxaBalance(len(self.eligible_addr_hashes))
}

func (self *Contract) GetBalanceIfEligible(address *common.Address) state_common.TaraxaBalance {
	return self.eligible_addr_hashes[keccak256.HashAndReturnByValue(address[:])]
}
