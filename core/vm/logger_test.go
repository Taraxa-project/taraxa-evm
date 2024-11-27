// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/holiman/uint256"
)

type account struct{}

func (account) Address() *common.Address            { return &common.Address{} }
func (account) GetBalance() *big.Int                { return nil }
func (account) GetNonce() *big.Int                  { return nil }
func (account) GetCodeHash() *common.Hash           { return nil }
func (account) GetCode() []byte                     { return []byte{} }
func (account) GetCodeSize() uint64                 { return 0 }
func (account) GetCommittedState(*big.Int) *big.Int { return nil }
func (account) GetState(*big.Int) *big.Int          { return nil }
func (account) HasSuicided() bool                   { return false }
func (account) IsNIL() bool                         { return true }
func (account) IsEIP161Empty() bool                 { return false }
func (account) SetCode([]byte)                      {}
func (account) AddBalance(*big.Int)                 {}
func (account) SubBalance(*big.Int)                 {}
func (account) IncrementNonce()                     {}
func (account) SetNonce(*big.Int)                   {}
func (account) SetState(*big.Int, *big.Int)         {}
func (account) Suicide(*common.Address)             {}

type dummyStatedb struct{}

func (*dummyStatedb) GetRefund() uint64                       { return 1337 }
func (*dummyStatedb) GetAccount(*common.Address) StateAccount { return account{} }
func (*dummyStatedb) AddLog(LogRecord)                        {}
func (*dummyStatedb) GetLogs() []LogRecord                    { return []LogRecord{} }
func (*dummyStatedb) AddRefund(uint64)                        {}
func (*dummyStatedb) SubRefund(uint64)                        {}
func (*dummyStatedb) RevertToSnapshot(int)                    {}
func (*dummyStatedb) Snapshot() int                           { return 0 }
func (*dummyStatedb) GetTransientState(addr *common.Address, key common.Hash) common.Hash {
	return common.Hash{}
}
func (*dummyStatedb) SetTransientState(addr *common.Address, key, value common.Hash) {}

func TestStoreCapture(t *testing.T) {
	var (
		evm    EVM
		logger = NewStructLogger(nil)
		mem    = NewMemory()
		stack  = newstack()
	)
	evm.Init(func(num types.BlockNum) *big.Int { panic("unexpected") }, &dummyStatedb{}, Opts{}, params.TestChainConfig, Config{})

	var code CodeAndHash
	code.Code = []byte{byte(PUSH1), 0x1, byte(PUSH1), 0x1, 0x0}
	contract := NewContract(CallFrame{account{}, account{}, nil, 10000, big.NewInt(0)}, code)

	stack.push(uint256.NewInt(1))
	stack.push(uint256.NewInt(0))
	var index common.Hash
	logger.CaptureState(&evm, 0, SSTORE, 0, 0, mem, stack, &contract, 0, nil)
	if len(logger.changedValues[*contract.Address()]) == 0 {
		t.Fatalf("expected exactly 1 changed value on address %x, got %d", contract.Address(), len(logger.changedValues[*contract.Address()]))
	}
	exp := common.BytesToHash(uint256.NewInt(1).Bytes())
	if logger.changedValues[*contract.Address()][index] != exp {
		t.Errorf("expected %x, got %x", exp, logger.changedValues[*contract.Address()][index])
	}
}
