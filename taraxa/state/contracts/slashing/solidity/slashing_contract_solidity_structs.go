// !!! Important: This file was was created manually with some parts generated automatically and copy pasted
//
// For automatic generation & copy paste struct:
//		 1. To generate ABI:
//			a) run `solc --abi --overwrite --optimize dpos_contract_interface.sol --output-dir .`
//			b) replace " by \" and copy&paste the ABI string into the TaraxaDposClientMetaData
//
//		 2. To generate solidity interface related structs:
//		 	a) run `abigen --abi=DposInterface.abi --pkg=taraxaDposClient --out=dpos_contract_interface.go`
//		    b) copy selected structs into this file
//
//		 3. a) remove generated file `rm DposInterface.abi`
// 		    b) remove generated file `rm dpos_contract_interface.go`

package sol

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

/*******************************************************/
/**** Automatically generated & Copy pasted structs ****/
/*******************************************************/

var TaraxaSlashingClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blocks\",\"type\":\"uint256\"}],\"name\":\"Jailed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"author\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"NewProof\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Slashed\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"author\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"vote1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vote2\",\"type\":\"bytes\"}],\"name\":\"commitDoubleVotingProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isJailed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

/*******************************************************/
/************** Manually created structs ***************/
/*******************************************************/

// !!! Important: arguments names inside "<...>Args" structs must match args names from solidity interface, otherwise it won't work

type CommitDoubleVotingProofArgs struct {
	Validator common.Address
	Vote1     []byte
	Vote2     []byte
}

type IsJailedArgs struct {
	Validator common.Address
}
