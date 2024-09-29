// !!! Important: This file was was created manually with some parts generated automatically and copy pasted
//
// For automatic generation & copy paste struct:
//		 1. To generate ABI:
//			a) run `solc --abi --overwrite --optimize dpos_contract_interface.sol --output-dir .`
//			b) replace " by \" and copy&paste the ABI string into the TaraxaDposClientMetaData
//		 2. To get implementation bytecode(TaraxaDposImplBytecode)
//			a) run `solc --bin-runtime --overwrite --optimize dpos_contract_impl.sol --output-dir .`
//			b) Copy bytecode from `DposDummyImpl.bin-runtime` file to `var TaraxaDposImplBytecode` variable in `dpos_contract_solidity_structs.go` file.
//		 2. To generate solidity interface related structs:
//		 	a) run `abigen --abi=DposInterface.abi --pkg=taraxaDposClient --out=dpos_contract_interface.go`
//		    b) copy selected structs into this file
//
//		 3. a) remove generated file `rm DposInterface.abi`
// 		    b) remove generated file `rm dpos_contract_interface.go`
// 		    c) remove generated file `rm DposDummyImpl.bin-runtime`

package dpos_sol

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

/*******************************************************/
/**** Automatically generated & Copy pasted structs ****/
/*******************************************************/

var TaraxaDposClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"CommissionRewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"CommissionSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Delegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Redelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateCanceledV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateConfirmed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateConfirmedV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Undelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegatedV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorInfoSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorRegistered\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"cancelUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"name\":\"cancelUndelegateV2\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimAllRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimCommissionRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"confirmUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"name\":\"confirmUndelegateV2\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getDelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"internalType\":\"struct DposInterface.DelegatorInfo\",\"name\":\"delegation\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.DelegationData[]\",\"name\":\"delegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"}],\"name\":\"getTotalDelegation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"total_delegation\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"name\":\"getUndelegationV2\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData\",\"name\":\"undelegation_data\",\"type\":\"tuple\"},{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"internalType\":\"struct DposInterface.UndelegationV2Data\",\"name\":\"undelegation_v2\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData[]\",\"name\":\"undelegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegationsV2\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData\",\"name\":\"undelegation_data\",\"type\":\"tuple\"},{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"internalType\":\"struct DposInterface.UndelegationV2Data[]\",\"name\":\"undelegations_v2\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidator\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"validator_info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidatorEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidators\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidatorsFor\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isValidatorEligible\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"validator_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"reDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vrf_key\",\"type\":\"bytes\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"registerValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"setCommission\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"setValidatorInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegateV2\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"undelegation_id\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// DO NOT CHANGE THIS VALUE IT WILL CAUSE HARDFORK
var TaraxaDposImplBytecode = common.Hex2Bytes("60806040526004361061011b575f3560e01c8063703812cc1161009d578063de8e4b5011610062578063de8e4b50146102f8578063ef5cfb8c146101d2578063f000322c1461030a578063f3094e9014610324578063fc5e7e091461033e575f80fd5b8063703812cc1461027f578063724ac6b01461029e5780638b49d394146102b8578063d0eebfe2146101d2578063d6fdc127146102e0575f80fd5b806345a02561116100e357806345a02561146101d25780634d99dd16146101ef5780634edd99431461020d5780635c19a95c1461023e578063618e38621461024c575f80fd5b806309b72e001461011f5780630babea4c146101535780631904bb2e1461017657806319d8024f146101a2578063399ff554146101d2575b5f80fd5b34801561012a575f80fd5b5061013e6101393660046103db565b505f90565b60405190151581526020015b60405180910390f35b34801561015e575f80fd5b5061017461016d366004610456565b5050505050565b005b348015610181575f80fd5b506101956101903660046104d1565b610366565b60405161014a91906105cc565b3480156101ad575f80fd5b506101c46101bc3660046103db565b60605f915091565b60405161014a9291906105de565b3480156101dd575f80fd5b506101746101ec3660046104d1565b50565b3480156101fa575f80fd5b50610174610209366004610664565b5050565b348015610218575f80fd5b5061023061022736600461068c565b506060915f9150565b60405161014a9291906106bd565b6101746101ec3660046104d1565b348015610257575f80fd5b506102666101393660046104d1565b60405167ffffffffffffffff909116815260200161014a565b34801561028a575f80fd5b5061017461029936600461073d565b505050565b3480156102a9575f80fd5b506101c461022736600461068c565b3480156102c3575f80fd5b506102d261022736600461068c565b60405161014a929190610776565b6101746102ee366004610890565b5050505050505050565b348015610303575f80fd5b505f610266565b348015610315575f80fd5b50610174610209366004610961565b34801561032f575f80fd5b5061013e6101393660046104d1565b348015610349575f80fd5b506103586101393660046104d1565b60405190815260200161014a565b6103c36040518061010001604052805f81526020015f81526020015f61ffff1681526020015f67ffffffffffffffff1681526020015f61ffff1681526020015f6001600160a01b0316815260200160608152602001606081525090565b919050565b803563ffffffff811681146103c3575f80fd5b5f602082840312156103eb575f80fd5b6103f4826103c8565b9392505050565b80356001600160a01b03811681146103c3575f80fd5b5f8083601f840112610421575f80fd5b50813567ffffffffffffffff811115610438575f80fd5b60208301915083602082850101111561044f575f80fd5b9250929050565b5f805f805f6060868803121561046a575f80fd5b610473866103fb565b9450602086013567ffffffffffffffff8082111561048f575f80fd5b61049b89838a01610411565b909650945060408801359150808211156104b3575f80fd5b506104c088828901610411565b969995985093965092949392505050565b5f602082840312156104e1575f80fd5b6103f4826103fb565b5f81518084525f5b8181101561050e576020818501810151868301820152016104f2565b505f602082860101526020601f19601f83011685010191505092915050565b5f610100825184526020830151602085015261ffff604084015116604085015267ffffffffffffffff60608401511660608501526080830151610576608086018261ffff169052565b5060a083015161059160a08601826001600160a01b03169052565b5060c08301518160c08601526105a9828601826104ea565b91505060e083015184820360e08601526105c382826104ea565b95945050505050565b602081525f6103f4602083018461052d565b5f60408083016040845280865180835260608601915060608160051b870101925060208089015f5b8381101561064e57888603605f19018552815180516001600160a01b0316875283015183870188905261063b8888018261052d565b9650509382019390820190600101610606565b5050961515959096019490945295945050505050565b5f8060408385031215610675575f80fd5b61067e836103fb565b946020939093013593505050565b5f806040838503121561069d575f80fd5b6106a6836103fb565b91506106b4602084016103c8565b90509250929050565b604080825283518282018190525f9190606090818501906020808901865b83811015610727578151805186528381015167ffffffffffffffff1684870152878101516001600160a01b031688870152860151151586860152608090940193908201906001016106db565b50505050851515602086015292506103f4915050565b5f805f6060848603121561074f575f80fd5b610758846103fb565b9250610766602085016103fb565b9150604084013590509250925092565b604080825283518282018190525f9190606090818501906020808901865b838110156107cb57815180516001600160a01b03168652830151805184870152830151878601529385019390820190600101610794565b505096151595909601949094525091949350505050565b634e487b7160e01b5f52604160045260245ffd5b5f82601f830112610805575f80fd5b813567ffffffffffffffff80821115610820576108206107e2565b604051601f8301601f19908116603f01168101908282118183101715610848576108486107e2565b81604052838152866020858801011115610860575f80fd5b836020870160208301375f602085830101528094505050505092915050565b803561ffff811681146103c3575f80fd5b5f805f805f805f8060c0898b0312156108a7575f80fd5b6108b0896103fb565b9750602089013567ffffffffffffffff808211156108cc575f80fd5b6108d88c838d016107f6565b985060408b01359150808211156108ed575f80fd5b6108f98c838d016107f6565b975061090760608c0161087f565b965060808b013591508082111561091c575f80fd5b6109288c838d01610411565b909650945060a08b0135915080821115610940575f80fd5b5061094d8b828c01610411565b999c989b5096995094979396929594505050565b5f8060408385031215610972575f80fd5b61097b836103fb565b91506106b46020840161087f56fea2646970667358221220224342bd6b858745c722aa7cd65ed118ec4fe04131b2432e5d0051eca5b98ffe64736f6c63430008170033")

// DposInterfaceDelegationData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceDelegationData struct {
	Account    common.Address
	Delegation DposInterfaceDelegatorInfo
}

// DposInterfaceDelegatorInfo is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceDelegatorInfo struct {
	Stake   *big.Int
	Rewards *big.Int
}

// DposInterfaceUndelegationData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceUndelegationData struct {
	Stake           *big.Int
	Block           uint64
	Validator       common.Address
	ValidatorExists bool
}

// DposInterfaceUndelegationV2Data is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceUndelegationV2Data struct {
	UndelegationData DposInterfaceUndelegationData
	UndelegationId   uint64
}

// DposInterfaceValidatorBasicInfo is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorBasicInfo struct {
	TotalStake           *big.Int
	CommissionReward     *big.Int
	Commission           uint16
	LastCommissionChange uint64
	UndelegationsCount   uint16
	Owner                common.Address
	Description          string
	Endpoint             string
}

// DposInterfaceValidatorData is an auto generated low-level Go binding around an user-defined struct.
type DposInterfaceValidatorData struct {
	Account common.Address
	Info    DposInterfaceValidatorBasicInfo
}

/*******************************************************/
/************** Manually created structs ***************/
/*******************************************************/

// !!! Important: arguments names inside "<...>Args" structs must match args names from solidity interface, otherwise it won't work

type UndelegateArgs struct {
	Validator common.Address
	Amount    *big.Int
}

type RedelegateArgs struct {
	ValidatorFrom common.Address
	ValidatorTo   common.Address
	Amount        *big.Int
}

type RegisterValidatorArgs struct {
	Validator   common.Address
	Proof       []byte
	VrfKey      []byte
	Commission  uint16
	Description string
	Endpoint    string
}
type SetValidatorInfoArgs struct {
	Validator   common.Address
	Description string
	Endpoint    string
}

type SetCommissionArgs struct {
	Validator  common.Address
	Commission uint16
}

type ConfirmUndelegateV2Args struct {
	Validator      common.Address
	UndelegationId uint64
}

type CancelUndelegateV2Args struct {
	Validator      common.Address
	UndelegationId uint64
}

type ValidatorAddressArgs struct {
	Validator common.Address
}

type ClaimAllRewardsArgs struct {
	Batch uint32
}

type GetValidatorsArgs struct {
	Batch uint32
}

type GetValidatorsForArgs struct {
	Owner common.Address
	Batch uint32
}

type GetTotalDelegationArgs struct {
	Delegator common.Address
}

type GetDelegationsArgs struct {
	Delegator common.Address
	Batch     uint32
}

type GetUndelegationsArgs struct {
	Delegator common.Address
	Batch     uint32
}

type GetUndelegationsV2Args struct {
	Delegator common.Address
	Batch     uint32
}

type GetUndelegationV2Args struct {
	Delegator      common.Address
	Validator      common.Address
	UndelegationId uint64
}
