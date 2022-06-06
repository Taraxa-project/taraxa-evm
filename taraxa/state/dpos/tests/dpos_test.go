package test_integration

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

func TestProof(t *testing.T) {
	pubkey, seckey := generateKeyPair()
	addr := common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
	proof, _ := sign(addr.Hash().Bytes(), seckey)
	pubkey2, err := crypto.Ecrecover(addr.Hash().Bytes(), append(proof[:64], proof[64]-27))
	if err != nil {
		t.Errorf(err.Error())
	}
	if !bytes.Equal(pubkey, pubkey2) {
		t.Errorf("pubkey mismatch: want: %x have: %x", pubkey, pubkey2)
	}
	if common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:]) != addr {
		t.Errorf("pubkey mismatch: want: %x have: %x", addr, addr)
	}
}

func TestRegisterValidator(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to register same validator twice
	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	// Try to register with not enough balance
	test.ExecuteAndCheck(validator2_owner, bigutil.Add(DefaultBalance, Big1), test.pack("registerValidator", validator2_addr, validator2_proof, uint16(10), "test", "test"), util.ErrorString(""), vm.ErrInsufficientBalanceForTransfer)
	// Try to register with wrong proof
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator2_proof, uint16(10), "test", "test"), dpos.ErrWrongProof, util.ErrorString(""))
}

func TestDelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to delegate to not existent validator
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("delegate", addr(2)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator2_addr, validator2_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	//Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))

	vali1_new_delegation := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))
	test.ExecuteAndCheck(validator1_owner, vali1_new_delegation, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Validator to does not exist
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, addr(3), DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Validator from does not exist
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", addr(3), validator1_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Non existen delegation
	test.ExecuteAndCheck(addr(3), Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	// InsufficientDelegation
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, bigutil.Add(vali1_new_delegation, Big1)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestUndelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	//Check from same undelegate request
	test.ExecuteAndCheck(val_owner, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrExistentUndelegation, util.ErrorString(""))
	// NonExistentValidator
	test.ExecuteAndCheck(val_owner, Big0, test.pack("undelegate", delegator_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// NonExistentDelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	// ErrInsufficientDelegation
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, bigutil.Add(DefaultMinimumDeposit, big.NewInt(1))), dpos.ErrInsufficientDelegation, util.ErrorString(""))
}

func TestRewardsAndCommission(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	txFee := bigutil.Div(TaraPrecision, big.NewInt(1000)) //  0.001 TARA

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)
	validator1_commission := uint16(500) // 5%
	delegator1_addr := validator1_owner
	delegator1_stake := DefaultMinimumDeposit

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)
	validator2_commission := uint16(200) // 2%
	delegator2_addr := validator2_owner
	delegator2_stake := DefaultMinimumDeposit

	delegator3_addr := addr(3)
	delegator3_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(3))

	/*
		Simulate scenario when we have:

		  - total unique txs count == 40
			- validator 1:
					- stake == 20% (from total stake)
					- he delegates to himself those 20%
					- added 8 unique txs
			- validator 2:
					- stake == 80% (from total stake)
					- he delegates to himself 20% (from total stake)
					- added 32 unique txs
			- delegator 1:
					- delegated 60% (from total stake) to validator 2


		After every participant claims his rewards:
			- validator1_rewards = (validator1_txs * blockReward) / total_txs
			- validator2_rewards = (validator2_txs * blockReward) / total_txs

			- delegator 1(validator 1) gets 100 % from validator1_rewards
			- delegator 2(validator 2) gets 25 % from validator2_rewards
			- delegator 3 gets 75 % from validator2_rewards
	*/

	// Creates validators & delegators
	test.ExecuteAndCheck(validator1_owner, delegator1_stake, test.pack("registerValidator", validator1_addr, validator1_proof, validator1_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake := delegator1_stake

	test.ExecuteAndCheck(validator2_owner, delegator2_stake, test.pack("registerValidator", validator2_addr, validator2_proof, validator2_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator2_stake)

	test.ExecuteAndCheck(delegator3_addr, delegator3_stake, test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator3_stake)

	// Simulated rewards statistics
	tmp_rewards_stats := rewards_stats.NewRewardsStats()
	fees_rewards := dpos.NewFeesRewards()

	validator1_stats := rewards_stats.ValidatorStats{}
	validator1_stats.UniqueTxsCount = 8
	validator1_stats.ValidCertVote = false
	initValidatorTxsStats(validator1_addr, &fees_rewards, txFee, validator1_stats.UniqueTxsCount)
	tmp_rewards_stats.ValidatorsStats[validator1_addr] = validator1_stats

	validator2_stats := rewards_stats.ValidatorStats{}
	validator2_stats.UniqueTxsCount = 32
	validator2_stats.ValidCertVote = false
	initValidatorTxsStats(validator2_addr, &fees_rewards, txFee, validator2_stats.UniqueTxsCount)
	tmp_rewards_stats.ValidatorsStats[validator2_addr] = validator2_stats

	tmp_rewards_stats.TotalUniqueTxsCount = validator1_stats.UniqueTxsCount + validator2_stats.UniqueTxsCount
	tmp_rewards_stats.TotalUniqueVotesCount = 0

	// Advance block
	test.AdvanceBlock(&tmp_rewards_stats, &fees_rewards)

	// Expected block reward
	expected_block_reward := bigutil.Mul(total_stake, big.NewInt(int64(test.Chain_cfg.DPOS.YieldPercentage)))
	expected_block_reward = bigutil.Div(expected_block_reward, bigutil.Mul(dpos.Big100, big.NewInt(int64(test.Chain_cfg.DPOS.BlocksPerYear))))

	// Expected participants rewards
	// validator1_rewards = (validator1_txs * blockReward) / total_txs
	validator1_total_reward := bigutil.Div(bigutil.Mul(expected_block_reward, big.NewInt(int64(validator1_stats.UniqueTxsCount))), big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount)))
	validator1_total_reward = bigutil.Add(validator1_total_reward, bigutil.Mul(txFee, big.NewInt(int64(validator1_stats.UniqueTxsCount))))
	expected_validator1_commission_reward := bigutil.Div(bigutil.Mul(validator1_total_reward, big.NewInt(int64(validator1_commission))), dpos.Big10000)
	expected_validator1_delegators_reward := bigutil.Sub(validator1_total_reward, expected_validator1_commission_reward)

	// validator2_rewards = (validator2_txs * blockReward) / total_txs
	validator2_total_reward := bigutil.Div(bigutil.Mul(expected_block_reward, big.NewInt(int64(validator2_stats.UniqueTxsCount))), big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount)))
	validator2_total_reward = bigutil.Add(validator2_total_reward, bigutil.Mul(txFee, big.NewInt(int64(validator2_stats.UniqueTxsCount))))
	expected_validator2_commission_reward := bigutil.Div(bigutil.Mul(validator2_total_reward, big.NewInt(int64(validator2_commission))), dpos.Big10000)
	expected_validator2_delegators_reward := bigutil.Sub(validator2_total_reward, expected_validator2_commission_reward)

	// delegator 1(validator 1) gets 100 % from validator1_rewards
	expected_delegator1_reward := expected_validator1_delegators_reward

	// delegator 2(validator 2) gets 25 % from validator2_rewards
	expected_delegator2_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(25)), dpos.Big100)

	// delegator 3 gets 75 % from validator2_rewards
	expected_delegator3_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(75)), dpos.Big100)

	// expected_block_rewardPlusFees := bigutil.Add(expected_block_reward, bigutil.Mul(txFee, big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount))))
	// expectedDelegatorsRewards := bigutil.Add(expected_delegator1_reward, bigutil.Add(expected_delegator2_reward, expected_delegator3_reward))
	// // Last digit is removed due to rounding error that makes these values unequal
	// tc.Assert.Equal(bigutil.Div(expected_block_rewardPlusFees, Big10), bigutil.Div(expectedDelegatorsRewards, Big10))

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("claimRewards", validator2_addr), dpos.ErrNonExistentDelegation, util.ErrorString(""))

	// Check delgators rewards
	delegator1_old_balance := test.GetBalance(delegator1_addr)
	delegator2_old_balance := test.GetBalance(delegator2_addr)
	delegator3_old_balance := test.GetBalance(delegator3_addr)

	test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("claimRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, Big0, test.pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator3_addr, Big0, test.pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))

	actual_delegator1_reward := bigutil.Sub(test.GetBalance(delegator1_addr), delegator1_old_balance)
	actual_delegator2_reward := bigutil.Sub(test.GetBalance(delegator2_addr), delegator2_old_balance)
	actual_delegator3_reward := bigutil.Sub(test.GetBalance(delegator3_addr), delegator3_old_balance)

	tc.Assert.Equal(expected_delegator1_reward, actual_delegator1_reward)
	tc.Assert.Equal(expected_delegator2_reward, actual_delegator2_reward)
	tc.Assert.Equal(expected_delegator3_reward, actual_delegator3_reward)

	// Check commission rewards
	validator1_old_balance := test.GetBalance(validator1_owner)
	validator2_old_balance := test.GetBalance(validator2_owner)

	test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("claimCommissionRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, Big0, test.pack("claimCommissionRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))

	actual_validator1_commission_reward := bigutil.Sub(test.GetBalance(validator1_owner), validator1_old_balance)
	actual_validator2_commission_reward := bigutil.Sub(test.GetBalance(validator2_owner), validator2_old_balance)

	tc.Assert.Equal(expected_validator1_commission_reward, actual_validator1_commission_reward)
	tc.Assert.Equal(expected_validator2_commission_reward, actual_validator2_commission_reward)
}

func TestGenesis(t *testing.T) {
	cfg := CopyDefaulChainConfig()

	delegator := addr(1)

	for i := uint64(0); i < 4; i++ {
		entry := dpos.GenesisValidator{addr(i), addr(i), 0, "", "", core.BalanceMap{}}
		entry.Delegations[delegator] = DefaultEligibilityBalanceThreshold
		cfg.DPOS.InitialValidators = append(cfg.DPOS.InitialValidators, entry)
	}
	accVoteCount := bigutil.Div(DefaultEligibilityBalanceThreshold, cfg.DPOS.VoteEligibilityBalanceStep)

	tc, test := init_test(t, cfg)

	defer test.end()

	totalAmountDelegated := bigutil.Mul(DefaultEligibilityBalanceThreshold, big.NewInt(4))

	tc.Assert.Equal(bigutil.Sub(DefaultBalance, totalAmountDelegated), test.GetBalance(addr(1)))
	tc.Assert.Equal(accVoteCount.Uint64()*4, test.GetDPOSReader().EligibleVoteCount())
	tc.Assert.Equal(totalAmountDelegated, test.GetDPOSReader().TotalAmountDelegated())
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(1)))
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(2)))
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(3)))
}

func TestSetCommissions(t *testing.T) {
	cfg := CopyDefaulChainConfig()
	cfg.DPOS.CommissionChangeDelta = 5
	cfg.DPOS.CommissionChangeFrequency = 4

	_, test := init_test(t, cfg)
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), Big0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(11)), util.ErrorString(""), util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(20)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(16)), util.ErrorString(""), util.ErrorString(""))
}

func TestDelegateMinMax(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Sub(DefaultMinimumDeposit, Big1), test.pack("delegate", val_addr), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(3), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestUndelegateMin(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), Big0, test.pack("undelegate", val_addr, Big1), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Mul(DefaultMinimumDeposit, big.NewInt(3)), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(addr(1), Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), Big0, test.pack("undelegate", val_addr, bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegateMinMax(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)

	init_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))

	test.ExecuteAndCheck(validator1_owner, init_stake, test.pack("registerValidator", validator1_addr, validator1_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, init_stake, test.pack("registerValidator", validator2_addr, validator2_proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, bigutil.Add(DefaultMinimumDeposit, Big1)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, bigutil.Sub(DefaultBalance, init_stake), test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(3), DefaultBalance, test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, Big1), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
}

// TODO undelegation test time wise
