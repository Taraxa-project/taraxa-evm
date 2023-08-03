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
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

/*******************************************************/
/**** Automatically generated & Copy pasted structs ****/
/*******************************************************/

var TaraxaSlashingClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"block\",\"type\":\"uint256\"}],\"name\":\"Jailed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"author\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"proof_type\",\"type\":\"uint8\"}],\"name\":\"NewProof\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Slashed\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vote1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vote2\",\"type\":\"bytes\"}],\"name\":\"commitDoubleVotingProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getDoubleVotingProofs\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"proof_author\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"block\",\"type\":\"uint256\"}],\"internalType\":\"struct SlashingInterface.DoubleVotingProof[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getJailInfo\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"jail_block\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"is_jailed\",\"type\":\"bool\"},{\"internalType\":\"uint32\",\"name\":\"proofs_count\",\"type\":\"uint32\"}],\"internalType\":\"struct SlashingInterface.JailInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getMaliciousValidators\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"jail_block\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"is_jailed\",\"type\":\"bool\"},{\"internalType\":\"uint32\",\"name\":\"proofs_count\",\"type\":\"uint32\"}],\"internalType\":\"struct SlashingInterface.JailInfo\",\"name\":\"jail_info\",\"type\":\"tuple\"}],\"internalType\":\"struct SlashingInterface.MaliciousValidator[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isJailed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// SlashingInterfaceDoubleVotingProof is an auto generated low-level Go binding around an user-defined struct.
type SlashingInterfaceDoubleVotingProof struct {
	ProofAuthor common.Address
	Block       *big.Int
}

// SlashingInterfaceJailInfo is an auto generated low-level Go binding around an user-defined struct.
type SlashingInterfaceJailInfo struct {
	JailBlock   *big.Int
	IsJailed    bool
	ProofsCount uint32
}

// SlashingInterfaceMaliciousValidator is an auto generated low-level Go binding around an user-defined struct.
type SlashingInterfaceMaliciousValidator struct {
	Validator common.Address
	JailInfo  SlashingInterfaceJailInfo
}

/*******************************************************/
/************** Manually created structs ***************/
/*******************************************************/

// !!! Important: arguments names inside "<...>Args" structs must match args names from solidity interface, otherwise it won't work

type CommitDoubleVotingProofArgs struct {
	Vote1 []byte
	Vote2 []byte
}

type ValidatorArg struct {
	Validator common.Address
}
