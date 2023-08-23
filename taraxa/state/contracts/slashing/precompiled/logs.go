package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

func checkError(log *vm.LogRecord, err error) *vm.LogRecord {
	if err != nil {
		panic("Update logs methods to correspond ABI: " + err.Error())
	}
	return log
}

type Logs struct {
	Events map[string]abi.Event
}

func (self *Logs) Init(events map[string]abi.Event) *Logs {
	self.Events = events

	return self
}

// All Make functions below are making log records for events.
// All hashes and data types should be the same as we have in solidity interface in ../solidity/slashing_contract_interface.sol
// If some event will be added or changed TestMakeLogsCheckTopics test should be modified

// event Jailed(address indexed validator, uint256 block)
func (self *Logs) MakeJailedLog(validator *common.Address, block uint64) vm.LogRecord {
	event := self.Events["Jailed"]

	return *checkError(event.MakeLog(slashing_contract_address, validator, block))
}
