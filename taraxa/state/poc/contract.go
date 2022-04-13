package poc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

var contract_address = new(common.Address).SetBytes(common.FromHex("0x00000000000000000000000000000000000000ee"))

var (
	field_staking_balances     = []byte{0}
)

var ErrTransferAmountIsZero = util.ErrorString("transfer amount is zero")
var ErrWithdrawalExceedsDeposit = util.ErrorString("withdrawal exceeds prior deposit value")
var ErrInsufficientBalanceForDeposits = util.ErrorString("insufficient balance for the deposits")
var ErrCallIsNotToplevel = util.ErrorString("only top-level calls are allowed")
var ErrNoTransfers = util.ErrorString("no transfers")
var ErrCallValueNonzero = util.ErrorString("call value must be zero")
var ErrDuplicateBeneficiary = util.ErrorString("duplicate beneficiary")

type Contract struct {
	storage                  StorageWrapper
	abi						 abi.ABI
}

type Addr2Balance = map[common.Address]*big.Int
type Addr2Addr2Balance = map[common.Address]Addr2Balance

const definition = `[
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "account",
				"type": "address"
			}
		],
		"name": "get_stake",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "account",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "stake",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	}
]`

func (self *Contract) Init(storage Storage) *Contract {
	self.storage.Init(storage)
	self.abi, _ = abi.JSON(strings.NewReader(definition))
	return self
}

func (self *Contract) Register(registry func(*common.Address, vm.PrecompiledContract)) {
	defensive_copy := *contract_address
	registry(&defensive_copy, self)
}

func (self *Contract) RequiredGas(ctx vm.CallFrame, evm *vm.EVM) uint64 {
	return uint64(len(ctx.Input)) * 20 // TODO
}

type GetStakeEvent struct {
	Account common.Address
}
type StakeEvenet struct {
	Account common.Address
	Amount *big.Int
}

func (self *Contract) Run(ctx vm.CallFrame, evm *vm.EVM) ([]byte, error) {
	if ctx.Value.Sign() != 0 {
		return nil, ErrCallValueNonzero
	}

	if evm.GetDepth() != 0 {
		return nil, ErrCallIsNotToplevel
	}

	method, err := self.abi.MethodById(ctx.Input)
	if err != nil {
		fmt.Println("XXXXXXXXXX ", err)
		return nil, nil
	}

	fmt.Println("XXXXXXXXXX " + method.Name)

	// First 4 bytes it method signature !!!!
	input := ctx.Input[4:]

	switch method.Name {
	case "stake":
		var data StakeEvenet
		if err = method.Inputs.Unpack(&data, input); err != nil {
			fmt.Println("XXXXXXXXXX ", err)
			return nil, nil
		}
		return nil, self.stake(data.Account, data.Amount)
	case "get_stake":
		var data GetStakeEvent
		if err = method.Inputs.Unpack(&data, input); err != nil {
			fmt.Println("XXXXXXXXXX ", err)
			return nil, nil
		}
		return method.Outputs.Pack(self.get_balance(data.Account))
	}
	return nil, nil
}

func (self *Contract) stake(acc common.Address, stake *big.Int) (err error) {
	if stake == nil || bigutil.IsZero(stake) {
		return ErrTransferAmountIsZero
	}
	balance := self.get_balance(acc)
	fmt.Println("XXXXXXXXXX stake ", stake)
	balance = bigutil.Add(balance, stake)
	fmt.Println("XXXXXXXXXX Add ", balance)
	self.put_balance(acc, balance)

	return
}

func (self *Contract) get_balance(addr common.Address) *big.Int {
	balance := bigutil.Big0
	balance_stor_k := stor_k_1(addr[:])
	fmt.Println("XXXXXXXXXX get_balance key ", balance_stor_k)
	self.storage.Get(balance_stor_k, func(bytes []byte) {
		balance = bigutil.FromBytes(bytes)
	})
	fmt.Println("XXXXXXXXXX balance ", balance)
	return balance
}

func (self *Contract) put_balance(addr common.Address, stake *big.Int) {
	fmt.Println("XXXXXXXXXX put_balance1 ", stake)
	balance_stor_k := stor_k_1(addr[:])
	fmt.Println("XXXXXXXXXX put_balance key ", balance_stor_k)
	self.storage.Put(balance_stor_k, stake.Bytes())
	balance := bigutil.Big0
	self.storage.Get(balance_stor_k, func(bytes []byte) {
		balance = bigutil.FromBytes(bytes)
	})
	fmt.Println("XXXXXXXXXX put_balance2 ", balance)
}