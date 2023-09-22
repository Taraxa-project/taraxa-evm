// !!! Important: This file was was created manually with some parts generated automatically and copy pasted
//
// For automatic generation & copy paste struct:
//		 1. To generate ABI:
//			a) run `solc --abi --overwrite --optimize slashing_contract_interface.sol --output-dir .`
//			b) replace " by \" and copy&paste the ABI string into the TaraxaSlashingClientMetaData
//
//		 2. To generate solidity interface related structs:
//		 	a) run `abigen --abi=SlashingInterface.abi --pkg=taraxaSlashingClient --out=slashing_contract_interface.go`
//		    b) copy selected structs into this file
//
//		 3. a) remove generated file `rm SlashingInterface.abi`
// 		    b) remove generated file `rm slashing_contract_interface.go`

package slashing_sol

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

/*******************************************************/
/**** Automatically generated & Copy pasted structs ****/
/*******************************************************/

var TaraxaSlashingClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"start_block\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"end_block\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"malicious_behaviour_type\",\"type\":\"uint8\"}],\"name\":\"Jailed\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vote_a\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vote_b\",\"type\":\"bytes\"}],\"name\":\"commitDoubleVotingProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getJailBlock\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

/*******************************************************/
/************** Manually created structs ***************/
/*******************************************************/

// !!! Important: arguments names inside "<...>Args" structs must match args names from solidity interface, otherwise it won't work

type CommitDoubleVotingProofArgs struct {
	VoteA []byte
	VoteB []byte
}

type ValidatorArg struct {
	Validator common.Address
}
