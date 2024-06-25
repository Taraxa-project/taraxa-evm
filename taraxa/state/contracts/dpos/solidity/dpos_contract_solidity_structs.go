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

var TaraxaDposClientMetaData = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"CommissionRewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"CommissionSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Delegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Redelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RewardsClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateCanceledV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateConfirmed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegateConfirmedV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Undelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"UndelegatedV2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorInfoSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"ValidatorRegistered\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"cancelUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"}],\"name\":\"cancelUndelegateV2\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimAllRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimCommissionRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"confirmUndelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"}],\"name\":\"confirmUndelegateV2\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getDelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"internalType\":\"struct DposInterface.DelegatorInfo\",\"name\":\"delegation\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.DelegationData[]\",\"name\":\"delegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"}],\"name\":\"getTotalDelegation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"total_delegation\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegations\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData[]\",\"name\":\"undelegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getUndelegationsV2\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"block\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"validator_exists\",\"type\":\"bool\"}],\"internalType\":\"struct DposInterface.UndelegationData\",\"name\":\"undelegation_data\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"undelegation_id\",\"type\":\"uint256\"}],\"internalType\":\"struct DposInterface.UndelegationV2Data[]\",\"name\":\"undelegations\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidator\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"validator_info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"getValidatorEligibleVotesCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidators\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"batch\",\"type\":\"uint32\"}],\"name\":\"getValidatorsFor\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total_stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"commission_reward\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"uint64\",\"name\":\"last_commission_change\",\"type\":\"uint64\"},{\"internalType\":\"uint16\",\"name\":\"undelegations_count\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"internalType\":\"struct DposInterface.ValidatorBasicInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"internalType\":\"struct DposInterface.ValidatorData[]\",\"name\":\"validators\",\"type\":\"tuple[]\"},{\"internalType\":\"bool\",\"name\":\"end\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"}],\"name\":\"isValidatorEligible\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"validator_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"reDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"vrf_key\",\"type\":\"bytes\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"registerValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"commission\",\"type\":\"uint16\"}],\"name\":\"setCommission\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"setValidatorInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

var TaraxaDposImplBytecode = common.Hex2Bytes("60806040526004361061013c575f3560e01c8063618e3862116100b3578063d6fdc1271161006d578063d6fdc12714610329578063de8e4b5014610341578063ef5cfb8c14610211578063f000322c14610353578063f3094e901461036d578063fc5e7e0914610387575f80fd5b8063618e38621461026d578063703812cc146102a0578063724ac6b0146102bf57806378df66e3146102d95780638b49d39414610301578063d0eebfe214610211575f80fd5b8063399ff55411610104578063399ff5541461021157806345a02561146102115780634d99dd16146101975780634edd99431461022e5780635774041d146101975780635c19a95c1461025f575f80fd5b806309b72e00146101405780630babea4c1461017457806311ce4015146101975780631904bb2e146101b557806319d8024f146101e1575b5f80fd5b34801561014b575f80fd5b5061015f61015a366004610424565b505f90565b60405190151581526020015b60405180910390f35b34801561017f575f80fd5b5061019561018e36600461049f565b5050505050565b005b3480156101a2575f80fd5b506101956101b136600461051a565b5050565b3480156101c0575f80fd5b506101d46101cf366004610542565b6103af565b60405161016b9190610628565b3480156101ec575f80fd5b506102036101fb366004610424565b60605f915091565b60405161016b92919061063a565b34801561021c575f80fd5b5061019561022b366004610542565b50565b348015610239575f80fd5b506102516102483660046106c0565b506060915f9150565b60405161016b929190610729565b61019561022b366004610542565b348015610278575f80fd5b5061028761015a366004610542565b60405167ffffffffffffffff909116815260200161016b565b3480156102ab575f80fd5b506101956102ba366004610782565b505050565b3480156102ca575f80fd5b506102036102483660046106c0565b3480156102e4575f80fd5b506102f36102483660046106c0565b60405161016b9291906107bb565b34801561030c575f80fd5b5061031b6102483660046106c0565b60405161016b929190610806565b610195610337366004610920565b5050505050505050565b34801561034c575f80fd5b505f610287565b34801561035e575f80fd5b506101956101b13660046109f1565b348015610378575f80fd5b5061015f61015a366004610542565b348015610392575f80fd5b506103a161015a366004610542565b60405190815260200161016b565b61040c6040518061010001604052805f81526020015f81526020015f61ffff1681526020015f67ffffffffffffffff1681526020015f61ffff1681526020015f6001600160a01b0316815260200160608152602001606081525090565b919050565b803563ffffffff8116811461040c575f80fd5b5f60208284031215610434575f80fd5b61043d82610411565b9392505050565b80356001600160a01b038116811461040c575f80fd5b5f8083601f84011261046a575f80fd5b50813567ffffffffffffffff811115610481575f80fd5b602083019150836020828501011115610498575f80fd5b9250929050565b5f805f805f606086880312156104b3575f80fd5b6104bc86610444565b9450602086013567ffffffffffffffff808211156104d8575f80fd5b6104e489838a0161045a565b909650945060408801359150808211156104fc575f80fd5b506105098882890161045a565b969995985093965092949392505050565b5f806040838503121561052b575f80fd5b61053483610444565b946020939093013593505050565b5f60208284031215610552575f80fd5b61043d82610444565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b5f610100825184526020830151602085015261ffff604084015116604085015267ffffffffffffffff606084015116606085015260808301516105d2608086018261ffff169052565b5060a08301516105ed60a08601826001600160a01b03169052565b5060c08301518160c08601526106058286018261055b565b91505060e083015184820360e086015261061f828261055b565b95945050505050565b602081525f61043d6020830184610589565b5f60408083016040845280865180835260608601915060608160051b870101925060208089015f5b838110156106aa57888603605f19018552815180516001600160a01b0316875283015183870188905261069788880182610589565b9650509382019390820190600101610662565b5050961515959096019490945295945050505050565b5f80604083850312156106d1575f80fd5b6106da83610444565b91506106e860208401610411565b90509250929050565b8051825260208082015167ffffffffffffffff16908301526040808201516001600160a01b0316908301526060908101511515910152565b604080825283519082018190525f906020906060840190828701845b8281101561076b576107588483516106f1565b6080939093019290840190600101610745565b505050809250505082151560208301529392505050565b5f805f60608486031215610794575f80fd5b61079d84610444565b92506107ab60208501610444565b9150604084013590509250925092565b604080825283519082018190525f906020906060840190828701845b8281101561076b5781516107ec8582516106f1565b850151608085015260a090930192908401906001016107d7565b604080825283518282018190525f9190606090818501906020808901865b8381101561085b57815180516001600160a01b03168652830151805184870152830151878601529385019390820190600101610824565b505096151595909601949094525091949350505050565b634e487b7160e01b5f52604160045260245ffd5b5f82601f830112610895575f80fd5b813567ffffffffffffffff808211156108b0576108b0610872565b604051601f8301601f19908116603f011681019082821181831017156108d8576108d8610872565b816040528381528660208588010111156108f0575f80fd5b836020870160208301375f602085830101528094505050505092915050565b803561ffff8116811461040c575f80fd5b5f805f805f805f8060c0898b031215610937575f80fd5b61094089610444565b9750602089013567ffffffffffffffff8082111561095c575f80fd5b6109688c838d01610886565b985060408b013591508082111561097d575f80fd5b6109898c838d01610886565b975061099760608c0161090f565b965060808b01359150808211156109ac575f80fd5b6109b88c838d0161045a565b909650945060a08b01359150808211156109d0575f80fd5b506109dd8b828c0161045a565b999c989b5096995094979396929594505050565b5f8060408385031215610a02575f80fd5b610a0b83610444565b91506106e86020840161090f56fea2646970667358221220e41e38fd4ca297782d3e987b2a6e4e7e679162f525272e4e7858214f81bdf2ee64736f6c63430008190033")

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
	UndelegationId   *big.Int
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
	UndelegationId *big.Int
}

type CancelUndelegateV2Args struct {
	Validator      common.Address
	UndelegationId *big.Int
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
