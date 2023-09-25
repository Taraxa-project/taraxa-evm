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

var TaraxaDposClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"CommissionRewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"CommissionSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Delegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Redelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateConfirmed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Undelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorInfoSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorRegistered\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"cancelUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimAllRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimCommissionRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"confirmUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getDelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"internalType\":\"struct DposInterface.DelegatorInfo\",\"name\":\"delegation\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.DelegationData[]\",\"name\":\"delegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"}],\"name\":\"getTotalDelegation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"total_delegation\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData[]\",\"name\":\"undelegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidator\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"validator_info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidatorEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidators\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidatorsFor\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isValidatorEligible\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"validator_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"reDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vrf_key\",\"type\":\"bytes\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"registerValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"setCommission\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"setValidatorInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Bytecode of compiled dpos_contract_impl.sol without the constructor code
var TaraxaDposImplBytecode = common.Hex2Bytes("60806040526004361061011e575f3560e01c8063703812cc1161009f578063de8e4b5011610063578063de8e4b50146103f2578063ef5cfb8c1461041c578063f000322c14610444578063f3094e901461046c578063fc5e7e09146104a85761011e565b8063703812cc1461030c578063724ac6b0146103345780638b49d39414610371578063d0eebfe2146103ae578063d6fdc127146103d65761011e565b806345a02561116100e657806345a02561146102275780634d99dd161461024f5780634edd9943146102775780635c19a95c146102b4578063618e3862146102d05761011e565b806309b72e00146101225780630babea4c1461015e5780631904bb2e1461018657806319d8024f146101c2578063399ff554146101ff575b5f80fd5b34801561012d575f80fd5b5061014860048036038101906101439190610611565b6104e4565b6040516101559190610656565b60405180910390f35b348015610169575f80fd5b50610184600480360381019061017f919061072a565b6104ea565b005b348015610191575f80fd5b506101ac60048036038101906101a791906107bb565b6104f1565b6040516101b99190610989565b60405180910390f35b3480156101cd575f80fd5b506101e860048036038101906101e39190610611565b6104fe565b6040516101f6929190610b52565b60405180910390f35b34801561020a575f80fd5b50610225600480360381019061022091906107bb565b610506565b005b348015610232575f80fd5b5061024d600480360381019061024891906107bb565b610509565b005b34801561025a575f80fd5b5061027560048036038101906102709190610baa565b61050c565b005b348015610282575f80fd5b5061029d60048036038101906102989190610be8565b610510565b6040516102ab929190610d30565b60405180910390f35b6102ce60048036038101906102c991906107bb565b61051a565b005b3480156102db575f80fd5b506102f660048036038101906102f191906107bb565b61051d565b6040516103039190610d6d565b60405180910390f35b348015610317575f80fd5b50610332600480360381019061032d9190610d86565b610523565b005b34801561033f575f80fd5b5061035a60048036038101906103559190610be8565b610528565b604051610368929190610b52565b60405180910390f35b34801561037c575f80fd5b5061039760048036038101906103929190610be8565b610532565b6040516103a5929190610ed8565b60405180910390f35b3480156103b9575f80fd5b506103d460048036038101906103cf91906107bb565b61053c565b005b6103f060048036038101906103eb9190611058565b61053f565b005b3480156103fd575f80fd5b50610406610549565b6040516104139190610d6d565b60405180910390f35b348015610427575f80fd5b50610442600480360381019061043d91906107bb565b61054d565b005b34801561044f575f80fd5b5061046a6004803603810190610465919061115a565b610550565b005b348015610477575f80fd5b50610492600480360381019061048d91906107bb565b610554565b60405161049f9190610656565b60405180910390f35b3480156104b3575f80fd5b506104ce60048036038101906104c991906107bb565b61055a565b6040516104db91906111a7565b60405180910390f35b5f919050565b5050505050565b6104f9610560565b919050565b60605f915091565b50565b50565b5050565b60605f9250929050565b50565b5f919050565b505050565b60605f9250929050565b60605f9250929050565b50565b5050505050505050565b5f90565b50565b5050565b5f919050565b5f919050565b6040518061010001604052805f81526020015f81526020015f61ffff1681526020015f67ffffffffffffffff1681526020015f61ffff1681526020015f73ffffffffffffffffffffffffffffffffffffffff16815260200160608152602001606081525090565b5f604051905090565b5f80fd5b5f80fd5b5f63ffffffff82169050919050565b6105f0816105d8565b81146105fa575f80fd5b50565b5f8135905061060b816105e7565b92915050565b5f60208284031215610626576106256105d0565b5b5f610633848285016105fd565b91505092915050565b5f8115159050919050565b6106508161063c565b82525050565b5f6020820190506106695f830184610647565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6106988261066f565b9050919050565b6106a88161068e565b81146106b2575f80fd5b50565b5f813590506106c38161069f565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f8401126106ea576106e96106c9565b5b8235905067ffffffffffffffff811115610707576107066106cd565b5b602083019150836001820283011115610723576107226106d1565b5b9250929050565b5f805f805f60608688031215610743576107426105d0565b5b5f610750888289016106b5565b955050602086013567ffffffffffffffff811115610771576107706105d4565b5b61077d888289016106d5565b9450945050604086013567ffffffffffffffff8111156107a05761079f6105d4565b5b6107ac888289016106d5565b92509250509295509295909350565b5f602082840312156107d0576107cf6105d0565b5b5f6107dd848285016106b5565b91505092915050565b5f819050919050565b6107f8816107e6565b82525050565b5f61ffff82169050919050565b610814816107fe565b82525050565b5f67ffffffffffffffff82169050919050565b6108368161081a565b82525050565b6108458161068e565b82525050565b5f81519050919050565b5f82825260208201905092915050565b5f5b83811015610882578082015181840152602081019050610867565b5f8484015250505050565b5f601f19601f8301169050919050565b5f6108a78261084b565b6108b18185610855565b93506108c1818560208601610865565b6108ca8161088d565b840191505092915050565b5f61010083015f8301516108eb5f8601826107ef565b5060208301516108fe60208601826107ef565b506040830151610911604086018261080b565b506060830151610924606086018261082d565b506080830151610937608086018261080b565b5060a083015161094a60a086018261083c565b5060c083015184820360c0860152610962828261089d565b91505060e083015184820360e086015261097c828261089d565b9150508091505092915050565b5f6020820190508181035f8301526109a181846108d5565b905092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f61010083015f8301516109e85f8601826107ef565b5060208301516109fb60208601826107ef565b506040830151610a0e604086018261080b565b506060830151610a21606086018261082d565b506080830151610a34608086018261080b565b5060a0830151610a4760a086018261083c565b5060c083015184820360c0860152610a5f828261089d565b91505060e083015184820360e0860152610a79828261089d565b9150508091505092915050565b5f604083015f830151610a9b5f86018261083c565b5060208301518482036020860152610ab382826109d2565b9150508091505092915050565b5f610acb8383610a86565b905092915050565b5f602082019050919050565b5f610ae9826109a9565b610af381856109b3565b935083602082028501610b05856109c3565b805f5b85811015610b405784840389528151610b218582610ac0565b9450610b2c83610ad3565b925060208a01995050600181019050610b08565b50829750879550505050505092915050565b5f6040820190508181035f830152610b6a8185610adf565b9050610b796020830184610647565b9392505050565b610b89816107e6565b8114610b93575f80fd5b50565b5f81359050610ba481610b80565b92915050565b5f8060408385031215610bc057610bbf6105d0565b5b5f610bcd858286016106b5565b9250506020610bde85828601610b96565b9150509250929050565b5f8060408385031215610bfe57610bfd6105d0565b5b5f610c0b858286016106b5565b9250506020610c1c858286016105fd565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b610c588161063c565b82525050565b608082015f820151610c725f8501826107ef565b506020820151610c85602085018261082d565b506040820151610c98604085018261083c565b506060820151610cab6060850182610c4f565b50505050565b5f610cbc8383610c5e565b60808301905092915050565b5f602082019050919050565b5f610cde82610c26565b610ce88185610c30565b9350610cf383610c40565b805f5b83811015610d23578151610d0a8882610cb1565b9750610d1583610cc8565b925050600181019050610cf6565b5085935050505092915050565b5f6040820190508181035f830152610d488185610cd4565b9050610d576020830184610647565b9392505050565b610d678161081a565b82525050565b5f602082019050610d805f830184610d5e565b92915050565b5f805f60608486031215610d9d57610d9c6105d0565b5b5f610daa868287016106b5565b9350506020610dbb868287016106b5565b9250506040610dcc86828701610b96565b9150509250925092565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b604082015f820151610e135f8501826107ef565b506020820151610e2660208501826107ef565b50505050565b606082015f820151610e405f85018261083c565b506020820151610e536020850182610dff565b50505050565b5f610e648383610e2c565b60608301905092915050565b5f602082019050919050565b5f610e8682610dd6565b610e908185610de0565b9350610e9b83610df0565b805f5b83811015610ecb578151610eb28882610e59565b9750610ebd83610e70565b925050600181019050610e9e565b5085935050505092915050565b5f6040820190508181035f830152610ef08185610e7c565b9050610eff6020830184610647565b9392505050565b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610f408261088d565b810181811067ffffffffffffffff82111715610f5f57610f5e610f0a565b5b80604052505050565b5f610f716105c7565b9050610f7d8282610f37565b919050565b5f67ffffffffffffffff821115610f9c57610f9b610f0a565b5b610fa58261088d565b9050602081019050919050565b828183375f83830152505050565b5f610fd2610fcd84610f82565b610f68565b905082815260208101848484011115610fee57610fed610f06565b5b610ff9848285610fb2565b509392505050565b5f82601f830112611015576110146106c9565b5b8135611025848260208601610fc0565b91505092915050565b611037816107fe565b8114611041575f80fd5b50565b5f813590506110528161102e565b92915050565b5f805f805f805f8060c0898b031215611074576110736105d0565b5b5f6110818b828c016106b5565b985050602089013567ffffffffffffffff8111156110a2576110a16105d4565b5b6110ae8b828c01611001565b975050604089013567ffffffffffffffff8111156110cf576110ce6105d4565b5b6110db8b828c01611001565b96505060606110ec8b828c01611044565b955050608089013567ffffffffffffffff81111561110d5761110c6105d4565b5b6111198b828c016106d5565b945094505060a089013567ffffffffffffffff81111561113c5761113b6105d4565b5b6111488b828c016106d5565b92509250509295985092959890939650565b5f80604083850312156111705761116f6105d0565b5b5f61117d858286016106b5565b925050602061118e85828601611044565b9150509250929050565b6111a1816107e6565b82525050565b5f6020820190506111ba5f830184611198565b9291505056fea26469706673582212204f0a28eea9ef996fe0e735dc8df5357a17933a8358e7d5639c30f6b28511a2eb64736f6c63430008160033")

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
	BlsKey      []byte
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

type UpdateBlsKeyArgs struct {
	Validator common.Address
	BlsKey    []byte
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
