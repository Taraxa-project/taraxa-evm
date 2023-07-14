// package slashing_tests

// import (
// 	"math/big"
// 	"testing"

// 	"github.com/Taraxa-project/taraxa-evm/core/vm"
// 	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
// 	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
// 	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
// 	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
// )

// // This strings should correspond to event signatures in ../solidity/dpos_contract_interface.sol file
// var DelegatedEventHash = *keccak256.Hash([]byte("Delegated(address,address,uint256)"))
// var UndelegatedEventHash = *keccak256.Hash([]byte("Undelegated(address,address,uint256)"))
// var UndelegateConfirmedEventHash = *keccak256.Hash([]byte("UndelegateConfirmed(address,address,uint256)"))
// var UndelegateCanceledEventHash = *keccak256.Hash([]byte("UndelegateCanceled(address,address,uint256)"))
// var RedelegatedEventHash = *keccak256.Hash([]byte("Redelegated(address,address,address,uint256)"))
// var RewardsClaimedEventHash = *keccak256.Hash([]byte("RewardsClaimed(address,address,uint256)"))
// var CommissionRewardsClaimedEventHash = *keccak256.Hash([]byte("CommissionRewardsClaimed(address,address,uint256)"))
// var CommissionSetEventHash = *keccak256.Hash([]byte("CommissionSet(address,uint16)"))
// var ValidatorRegisteredEventHash = *keccak256.Hash([]byte("ValidatorRegistered(address)"))
// var ValidatorInfoSetEventHash = *keccak256.Hash([]byte("ValidatorInfoSet(address)"))

// type IsJailedRet struct {
// 	End bool
// }

// func TestCommitDoubleVotingProof(t *testing.T) {
// 	_, test := init_test(t, CopyDefaultChainConfig())
// 	defer test.end()

// 	validator1_owner := addr(1)
// 	validator1_addr, validator1_proof := generateAddrAndProof()

// 	validator2_owner := addr(2)
// 	validator2_addr, validator2_proof := generateAddrAndProof()

// 	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
// 	test.CheckContractBalance(DefaultMinimumDeposit)
// 	// Try to register same validator twice
// 	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
// 	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
// 	// Try to register with not enough balance
// 	test.ExecuteAndCheck(validator2_owner, bigutil.Add(DefaultBalance, big.NewInt(1)), test.pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), vm.ErrInsufficientBalanceForTransfer)
// 	// Try to register with wrong proof
// 	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrWrongProof, util.ErrorString(""))
// }

// // func TestProof(t *testing.T) {
// // 	pubkey, seckey := generateKeyPair()
// // 	addr := common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
// // 	proof, _ := sign(addr.Hash().Bytes(), seckey)
// // 	pubkey2, err := crypto.Ecrecover(addr.Hash().Bytes(), append(proof[:64], proof[64]-27))
// // 	if err != nil {
// // 		t.Errorf(err.Error())
// // 	}
// // 	if !bytes.Equal(pubkey, pubkey2) {
// // 		t.Errorf("pubkey mismatch: want: %x have: %x", pubkey, pubkey2)
// // 	}
// // 	if common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:]) != addr {
// // 		t.Errorf("pubkey mismatch: want: %x have: %x", addr, addr)
// // 	}
// // }
