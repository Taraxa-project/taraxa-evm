package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

func getEventHash(str string) *common.Hash {
	return keccak256.Hash([]byte(str))
}

// event Delegated(address indexed delegator, address indexed validator, uint256 amount);
var DelegatedEventHash = getEventHash("Delegated(address,address,uint256)")

func MakeDelegatedLog(delegator, validator *common.Address, amount *big.Int) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *DelegatedEventHash
	topics[1] = delegator.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: amount.Bytes()}
}

// event Undelegated(address indexed delegator, address indexed validator, uint256 amount);
var UndelegatedEventHash = getEventHash("Undelegated(address,address,uint256)")

func MakeUndelegatedLog(delegator, validator *common.Address, amount *big.Int) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *UndelegatedEventHash
	topics[1] = delegator.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: amount.Bytes()}
}

// event UndelegateConfirmed(address indexed delegator, address indexed validator, uint256 amount);
var UndelegateConfirmedEventHash = getEventHash("UndelegateConfirmed(address,uint256)")

func MakeUndelegateConfirmedLog(delegator, validator *common.Address, amount *big.Int) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *UndelegateConfirmedEventHash
	topics[1] = delegator.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: amount.Bytes()}
}

// event UndelegateCanceled(address indexed delegator, address indexed validator, uint256 amount);
var UndelegateCanceledEventHash = getEventHash("UndelegateCanceled(address,uint256)")

func MakeUndelegateCanceledLog(delegator, validator *common.Address, amount *big.Int) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *UndelegateCanceledEventHash
	topics[1] = delegator.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: amount.Bytes()}
}

// event Redelegated(address indexed delegator, address indexed from, address indexed to, uint256 amount);
var RedelegatedEventHash = getEventHash("UndelegateCanceled(address,address,address,uint256)")

func MakeRedelegatedLog(delegator, from, to *common.Address, amount *big.Int) vm.LogRecord {
	topics := make([]common.Hash, 4)
	topics[0] = *RedelegatedEventHash
	topics[1] = delegator.Hash()
	topics[2] = from.Hash()
	topics[3] = to.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: amount.Bytes()}
}

// event RewardsClaimed(address indexed account, address indexed validator);
var RewardsClaimedEventHash = getEventHash("RewardsClaimed(address,address)")

func MakeRewardsClaimedLog(account, validator *common.Address) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *RewardsClaimedEventHash
	topics[1] = account.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: []byte{}}
}

// event CommissionRewardsClaimed(address indexed account, address indexed validator);
var ComissionRewardsClaimedEventHash = getEventHash("CommissionRewardsClaimed(address,address)")

func MakeComissionRewardsClaimedLog(account, validator *common.Address) vm.LogRecord {
	topics := make([]common.Hash, 3)
	topics[0] = *ComissionRewardsClaimedEventHash
	topics[1] = account.Hash()
	topics[2] = validator.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: []byte{}}
}

// event CommissionSet(address indexed validator, uint16 comission);
var CommissionSetEventHash = getEventHash("CommissionSet(address,uint16)")

func MakeCommissionSetLog(account *common.Address, amount uint16) vm.LogRecord {
	topics := make([]common.Hash, 2)
	topics[0] = *CommissionSetEventHash
	topics[1] = account.Hash()

	big_amount := big.NewInt(int64(amount))
	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: big_amount.Bytes()}
}

// event ValidatorRegistered(address indexed validator);
var ValidatorRegisteredEventHash = getEventHash("ValidatorRegistered(address)")

func MakeValidatorRegisteredLog(account *common.Address) vm.LogRecord {
	topics := make([]common.Hash, 2)
	topics[0] = *ValidatorRegisteredEventHash
	topics[1] = account.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: []byte{}}
}

// event ValidatorInfoSet(address indexed validator);
var ValidatorInfoSetEventHash = getEventHash("ValidatorInfoSet(address)")

func MakeValidatorInfoSetLog(account *common.Address) vm.LogRecord {
	topics := make([]common.Hash, 2)
	topics[0] = *ValidatorInfoSetEventHash
	topics[1] = account.Hash()

	return vm.LogRecord{Address: *contract_address, Topics: topics, Data: []byte{}}
}
