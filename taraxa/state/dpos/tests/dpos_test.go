package test_integration

import (
	"bytes"
	"fmt"
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

	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to register same validator twice
	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	// Try to register with not enough balance
	test.ExecuteAndCheck(validator2_owner, bigutil.Add(DefaultBalance, Big1), test.pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), vm.ErrInsufficientBalanceForTransfer)
	// Try to register with wrong proof
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrWrongProof, util.ErrorString(""))
}

func TestDelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to delegate to not existent validator
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("delegate", addr(2)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestDelegateMinMax(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Sub(DefaultMinimumDeposit, Big1), test.pack("delegate", val_addr), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(3), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Sub(DefaultBalance, DefaultMinimumDeposit), test.pack("delegate", val_addr), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	//Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))

	vali1_new_delegation := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))
	test.ExecuteAndCheck(validator1_owner, vali1_new_delegation, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
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

func TestRedelegateMinMax(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)

	init_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))

	test.ExecuteAndCheck(validator1_owner, init_stake, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, init_stake, test.pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, bigutil.Add(DefaultMinimumDeposit, Big1)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, bigutil.Sub(DefaultBalance, init_stake), test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(3), DefaultBalance, test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("reDelegate", validator1_addr, validator2_addr, Big1), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
}

func TestUndelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
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

func TestConfirmUndelegate(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("confirmUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// ErrLockedUndelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("confirmUndelegate", val_addr), dpos.ErrLockedUndelegation, util.ErrorString(""))

	// Advance 2 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)

	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("confirmUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
}

func TestCancelUndelegate(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("cancelUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))

	// Undelegate and check if validator's total stake was increased
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw := test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// Undelegate and check if validator's total stake was decreased
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(DefaultMinimumDeposit, validator.ValidatorInfo.TotalStake)

	// Cancel undelegate and check if validator's total stake was increased again
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("cancelUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("cancelUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
}

func TestUndelegateMin(t *testing.T) {
	_, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), Big0, test.pack("undelegate", val_addr, Big1), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Mul(DefaultMinimumDeposit, big.NewInt(3)), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(addr(1), Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), Big0, test.pack("undelegate", val_addr, bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))), util.ErrorString(""), util.ErrorString(""))
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
	delegator3_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(4))

	validator4_addr, validator4_proof := generateAddrAndProof()
	validator4_owner := addr(4)
	validator4_commission := uint16(0) // 0%
	delegator4_addr := validator4_owner
	delegator4_stake := DefaultMinimumDeposit

	validator5_addr, validator5_proof := generateAddrAndProof()
	validator5_owner := addr(5)
	validator5_commission := uint16(10000) //100%
	// delegator5_addr := validator5_owner
	delegator5_stake := DefaultMinimumDeposit

	/*
		Simulate scenario when we have:

		  - total unique txs count == 40
			- validator 1:
					- stake == 12.5% (from total stake)
					- he delegates to himself those 12.5%
					- added 8 unique txs
					- 1 vote
			- validator 2:
					- stake == 62.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- added 32 unique txs
					- 1 vote
			- delegator 3:
					- delegated 50% (from total stake) to validator 2
			- validator 4:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote
			- validator 5:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote

		After every participant claims his rewards:

			- block author (validator 1) - added 7 votes => bonus 1 vote
			- delegator 1(validator 1) gets 100 % from validator1_rewards
			- delegator 2(validator 2) gets 20 % from validator2_rewards
			- delegator 3 gets 80 % from validator2_rewards
			- delegator 4 gets 100% reward for 1 vote
	*/

	// Creates validators & delegators
	test.ExecuteAndCheck(validator1_owner, delegator1_stake, test.pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, validator1_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake := delegator1_stake

	test.ExecuteAndCheck(validator2_owner, delegator2_stake, test.pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, validator2_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator2_stake)

	test.ExecuteAndCheck(delegator3_addr, delegator3_stake, test.pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator3_stake)

	test.ExecuteAndCheck(validator4_owner, delegator4_stake, test.pack("registerValidator", validator4_addr, validator4_proof, DefaultVrfKey, validator4_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator4_stake)

	test.ExecuteAndCheck(validator5_owner, delegator5_stake, test.pack("registerValidator", validator5_addr, validator5_proof, DefaultVrfKey, validator5_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator5_stake)

	// Simulated rewards statistics
	tmp_rewards_stats := rewards_stats.NewRewardsStats()
	fees_rewards := dpos.NewFeesRewards()

	validator1_stats := rewards_stats.ValidatorStats{}
	validator1_stats.UniqueTxsCount = 8
	validator1_stats.VoteWeight = 1
	initValidatorTxsStats(validator1_addr, &fees_rewards, txFee, validator1_stats.UniqueTxsCount)
	tmp_rewards_stats.ValidatorsStats[validator1_addr] = validator1_stats

	validator2_stats := rewards_stats.ValidatorStats{}
	validator2_stats.UniqueTxsCount = 32
	validator2_stats.VoteWeight = 5
	initValidatorTxsStats(validator2_addr, &fees_rewards, txFee, validator2_stats.UniqueTxsCount)
	tmp_rewards_stats.ValidatorsStats[validator2_addr] = validator2_stats

	validator4_stats := rewards_stats.ValidatorStats{}
	validator4_stats.VoteWeight = 1
	tmp_rewards_stats.ValidatorsStats[validator4_addr] = validator4_stats

	tmp_rewards_stats.TotalUniqueTxsCount = validator1_stats.UniqueTxsCount + validator2_stats.UniqueTxsCount
	tmp_rewards_stats.TotalVotesWeight = 7
	tmp_rewards_stats.MaxVotesWeight = 8

	// Advance block
	test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats, &fees_rewards)

	// Expected block reward
	expected_block_reward := bigutil.Mul(total_stake, big.NewInt(int64(test.Chain_cfg.DPOS.YieldPercentage)))
	expected_block_reward = bigutil.Div(expected_block_reward, bigutil.Mul(dpos.Big100, big.NewInt(int64(test.Chain_cfg.DPOS.BlocksPerYear))))

	// Spliting block rewards between votes and blocks
	expected_trx_reward := bigutil.Div(bigutil.Mul(expected_block_reward, dpos.VotesToTrasnactionsRatio), dpos.Big100)
	expected_vote_reward := bigutil.Sub(expected_block_reward, expected_trx_reward)

	// Vote bonus rewards - aka Author reward
	// MaxBlockAuthorReward = 10
	bonus_reward := bigutil.Div(bigutil.Mul(expected_vote_reward, big.NewInt(int64(10))), dpos.Big100)
	expected_vote_reward = bigutil.Sub(expected_vote_reward, bonus_reward)

	// Vote bonus rewards - aka Author reward
	max_votes_weigh := dpos.Max(tmp_rewards_stats.MaxVotesWeight, tmp_rewards_stats.TotalVotesWeight)
	two_t_plus_one := max_votes_weigh*2/3 + 1
	author_reward := bigutil.Div(bigutil.Mul(bonus_reward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight-two_t_plus_one))), big.NewInt(int64(max_votes_weigh-two_t_plus_one)))

	// Expected participants rewards
	// validator1_rewards = (validator1_txs * blockReward) / total_txs
	validator1_total_reward := bigutil.Div(bigutil.Mul(expected_trx_reward, big.NewInt(int64(validator1_stats.UniqueTxsCount))), big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount)))
	validator1_total_reward = bigutil.Add(validator1_total_reward, bigutil.Mul(txFee, big.NewInt(int64(validator1_stats.UniqueTxsCount))))
	// Add vote reward
	validatorVoteReward := bigutil.Mul(big.NewInt(int64(validator1_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator1_total_reward = bigutil.Add(validator1_total_reward, validatorVoteReward)
	// Commission reward
	expected_validator1_commission_reward := bigutil.Div(bigutil.Mul(validator1_total_reward, big.NewInt(int64(validator1_commission))), dpos.Big10000)
	expected_validator1_delegators_reward := bigutil.Sub(validator1_total_reward, expected_validator1_commission_reward)
	// Add author reward
	author_commission_reward := bigutil.Div(bigutil.Mul(author_reward, big.NewInt(int64(validator1_commission))), dpos.Big10000)
	author_reward = bigutil.Sub(author_reward, author_commission_reward)
	expected_validator1_delegators_reward = bigutil.Add(expected_validator1_delegators_reward, author_reward)
	expected_validator1_commission_reward = bigutil.Add(expected_validator1_commission_reward, author_commission_reward)

	// validator2_rewards = (validator2_txs * blockReward) / total_txs
	validator2_total_reward := bigutil.Div(bigutil.Mul(expected_trx_reward, big.NewInt(int64(validator2_stats.UniqueTxsCount))), big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount)))
	validator2_total_reward = bigutil.Add(validator2_total_reward, bigutil.Mul(txFee, big.NewInt(int64(validator2_stats.UniqueTxsCount))))
	// Add vote reward
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator2_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator2_total_reward = bigutil.Add(validator2_total_reward, validatorVoteReward)

	expected_validator2_commission_reward := bigutil.Div(bigutil.Mul(validator2_total_reward, big.NewInt(int64(validator2_commission))), dpos.Big10000)
	expected_validator2_delegators_reward := bigutil.Sub(validator2_total_reward, expected_validator2_commission_reward)

	// Add vote reward for validator 4
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator4_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	expected_delegator4_reward := validatorVoteReward

	// delegator 1(validator 1) gets 100 % from validator1_rewards
	expected_delegator1_reward := expected_validator1_delegators_reward

	// delegator 2(validator 2) gets 25 % from validator2_rewards
	expected_delegator2_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(20)), dpos.Big100)

	// delegator 3 gets 75 % from validator2_rewards
	expected_delegator3_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(80)), dpos.Big100)

	// expected_trx_rewardPlusFees := bigutil.Add(expected_trx_reward, bigutil.Mul(txFee, big.NewInt(int64(tmp_rewards_stats.TotalUniqueTxsCount))))
	// expectedDelegatorsRewards := bigutil.Add(expected_delegator1_reward, bigutil.Add(expected_delegator2_reward, expected_delegator3_reward))
	// // Last digit is removed due to rounding error that makes these values unequal
	// tc.Assert.Equal(bigutil.Div(expected_trx_rewardPlusFees, Big10), bigutil.Div(expectedDelegatorsRewards, Big10))

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(validator1_owner, Big0, test.pack("claimRewards", validator2_addr), dpos.ErrNonExistentDelegation, util.ErrorString(""))

	// Check delgators rewards
	delegator1_old_balance := test.GetBalance(delegator1_addr)
	delegator2_old_balance := test.GetBalance(delegator2_addr)
	delegator3_old_balance := test.GetBalance(delegator3_addr)
	delegator4_old_balance := test.GetBalance(delegator4_addr)

	test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("claimRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, Big0, test.pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator3_addr, Big0, test.pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator4_addr, Big0, test.pack("claimRewards", validator4_addr), util.ErrorString(""), util.ErrorString(""))

	actual_delegator1_reward := bigutil.Sub(test.GetBalance(delegator1_addr), delegator1_old_balance)
	actual_delegator2_reward := bigutil.Sub(test.GetBalance(delegator2_addr), delegator2_old_balance)
	actual_delegator3_reward := bigutil.Sub(test.GetBalance(delegator3_addr), delegator3_old_balance)
	actual_delegator4_reward := bigutil.Sub(test.GetBalance(delegator4_addr), delegator4_old_balance)

	tc.Assert.Equal(expected_delegator1_reward, actual_delegator1_reward)
	tc.Assert.Equal(expected_delegator2_reward, actual_delegator2_reward)
	tc.Assert.Equal(expected_delegator3_reward, actual_delegator3_reward)
	tc.Assert.Equal(expected_delegator4_reward, actual_delegator4_reward)

	// Check commission rewards
	validator1_old_balance := test.GetBalance(validator1_owner)
	validator2_old_balance := test.GetBalance(validator2_owner)
	validator4_old_balance := test.GetBalance(validator4_owner)

	test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("claimCommissionRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, Big0, test.pack("claimCommissionRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator4_addr, Big0, test.pack("claimCommissionRewards", validator4_addr), util.ErrorString(""), util.ErrorString(""))

	actual_validator1_commission_reward := bigutil.Sub(test.GetBalance(validator1_owner), validator1_old_balance)
	actual_validator2_commission_reward := bigutil.Sub(test.GetBalance(validator2_owner), validator2_old_balance)
	actual_validator4_commission_reward := bigutil.Sub(test.GetBalance(validator4_owner), validator4_old_balance)

	tc.Assert.Equal(expected_validator1_commission_reward, actual_validator1_commission_reward)
	tc.Assert.Equal(expected_validator2_commission_reward, actual_validator2_commission_reward)
	tc.Assert.Equal(Big0.Cmp(actual_validator4_commission_reward), 0)
}

func TestGenesis(t *testing.T) {
	cfg := CopyDefaulChainConfig()

	delegator := addr(1)

	for i := uint64(1); i < 5; i++ {
		entry := dpos.GenesisValidator{addr(i), addr(i), DefaultVrfKey, 0, "", "", core.BalanceMap{}}
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

func TestSetValidatorInfo(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))

	validator_raw := test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal("test_description", validator.ValidatorInfo.Description)
	tc.Assert.Equal("test_endpoint", validator.ValidatorInfo.Endpoint)

	// Change description & endpoint and see it getValidator returns changed values
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setValidatorInfo", val_addr, "modified_description", "modified_endpoint"), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal("modified_description", validator.ValidatorInfo.Description)
	tc.Assert.Equal("modified_endpoint", validator.ValidatorInfo.Endpoint)

	// Try to set invalid(too long) description & endpoint
	invalid_description := "100+char_description................................................................................."
	tc.Assert.Greater(len(invalid_description), dpos.MaxDescriptionLength)
	// ErrMaxDescriptionLengthExceeded
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setValidatorInfo", addr(2), invalid_description, "modified_endpoint"), dpos.ErrMaxDescriptionLengthExceeded, util.ErrorString(""))

	invalid_endpoint := "100+char_endpoint.................................."
	tc.Assert.Greater(len(invalid_endpoint), dpos.MaxEndpointLength)
	// ErrMaxEndpointLengthExceeded
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setValidatorInfo", addr(2), "modified_description", invalid_endpoint), dpos.ErrMaxEndpointLengthExceeded, util.ErrorString(""))

	// ErrWrongOwnerAcc
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setValidatorInfo", addr(2), "modified_description", "modified_endpoint"), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
}

func TestSetCommission(t *testing.T) {
	cfg := CopyDefaulChainConfig()
	cfg.DPOS.CommissionChangeDelta = 5
	cfg.DPOS.CommissionChangeFrequency = 4

	_, test := init_test(t, cfg)
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), Big0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)

	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(11)), util.ErrorString(""), util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)

	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(20)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, Big0, test.pack("setCommission", val_addr, uint16(16)), util.ErrorString(""), util.ErrorString(""))
}

func TestGetValidators(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3*dpos.GetValidatorsMaxCount - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := CopyDefaulChainConfig()
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = DefaultBalance
	}
	tc, test := init_test(t, cfg)
	defer test.end()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21400
	intristic_gas_batch1 := 21464
	intristic_gas_batch2 := 21464
	intristic_gas_batch3 := 21464

	// Get first batch of validators from contract
	batch0_result := test.ExecuteAndCheck(addr(1), Big0, test.pack("getValidators", uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetValidatorsRet)
	test.unpack(batch0_parsed_result, "getValidators", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch0_parsed_result.Validators))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(dpos.GetValidatorsMaxCount)+"_description", batch0_parsed_result.Validators[len(batch0_parsed_result.Validators)-1].Info.Description)

	// Get second batch of validators from contract
	batch1_result := test.ExecuteAndCheck(addr(1), Big0, test.pack("getValidators", uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetValidatorsRet)
	test.unpack(batch1_parsed_result, "getValidators", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch1_parsed_result.Validators))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(2*dpos.GetValidatorsMaxCount)+"_description", batch1_parsed_result.Validators[len(batch1_parsed_result.Validators)-1].Info.Description)

	// Get third batch of validators from contract
	batch2_result := test.ExecuteAndCheck(addr(1), Big0, test.pack("getValidators", uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetValidatorsRet)
	test.unpack(batch2_parsed_result, "getValidators", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetValidatorsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount - 1
	tc.Assert.Equal(dpos.GetValidatorsMaxCount-1, len(batch2_parsed_result.Validators))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(3*dpos.GetValidatorsMaxCount-1)+"_description", batch2_parsed_result.Validators[len(batch2_parsed_result.Validators)-1].Info.Description)

	// Get fourth batch of validators from contract - it should return 0 validators
	batch3_result := test.ExecuteAndCheck(addr(1), Big0, test.pack("getValidators", uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetValidatorsRet)
	test.unpack(batch3_parsed_result, "getValidators", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned validators is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Validators))
	tc.Assert.Equal(true, batch3_parsed_result.End)
}

func TestGetDelegations(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3 * dpos.GetDelegationsMaxCount
	gen_delegator1_delegations := gen_validators_num - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := CopyDefaulChainConfig()
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = DefaultBalance
	}

	// Generate  delegator and set some balance to him
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = DefaultBalance

	tc, test := init_test(t, cfg)
	defer test.end()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator delegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.pack("delegate", gen_validators[i].address), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21592
	intristic_gas_batch1 := 21656
	intristic_gas_batch2 := 21656
	intristic_gas_batch3 := 21656

	// Get first batch of delegator1 delegations from contract
	batch0_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getDelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetDelegationsRet)
	test.unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetDelegationsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned delegations is == dpos.GetDelegationsMaxCount
	tc.Assert.Equal(dpos.GetDelegationsMaxCount, len(batch0_parsed_result.Delegations))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[len(batch0_parsed_result.Delegations)-1].address, batch0_parsed_result.Delegations[len(batch0_parsed_result.Delegations)-1].Account)

	// Get second batch of delegator1 delegations from contract
	batch1_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getDelegations", delegator1_addr, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetDelegationsRet)
	test.unpack(batch1_parsed_result, "getDelegations", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetDelegationsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned delegations is == dpos.GetDelegationsMaxCount
	tc.Assert.Equal(dpos.GetDelegationsMaxCount, len(batch1_parsed_result.Delegations))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[dpos.GetDelegationsMaxCount+len(batch1_parsed_result.Delegations)-1].address, batch1_parsed_result.Delegations[len(batch1_parsed_result.Delegations)-1].Account)

	// Get third batch of delegator1 delegations from contract
	batch2_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getDelegations", delegator1_addr, uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetDelegationsRet)
	test.unpack(batch2_parsed_result, "getDelegations", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetDelegationsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned v is == dpos.GetDelegationsMaxCount - 1
	tc.Assert.Equal(dpos.GetDelegationsMaxCount-1, len(batch2_parsed_result.Delegations))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[2*dpos.GetDelegationsMaxCount+len(batch2_parsed_result.Delegations)-1].address, batch2_parsed_result.Delegations[len(batch2_parsed_result.Delegations)-1].Account)

	// Get fourth batch of delegator1 delegations from contract - it should return 0 delegations
	batch3_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getDelegations", delegator1_addr, uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetDelegationsRet)
	test.unpack(batch3_parsed_result, "getDelegations", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned delegations is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Delegations))
	tc.Assert.Equal(true, batch3_parsed_result.End)
}

func TestGetUndelegations(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3 * dpos.GetUndelegationsMaxCount
	gen_delegator1_delegations := gen_validators_num - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := CopyDefaulChainConfig()
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = DefaultBalance
	}

	// Generate  delegator and set some balance to him
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = DefaultBalance

	tc, test := init_test(t, cfg)
	defer test.end()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator delegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.pack("delegate", gen_validators[i].address), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator undelegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.pack("undelegate", gen_validators[i].address, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21592
	intristic_gas_batch1 := 21656
	intristic_gas_batch2 := 21656
	intristic_gas_batch3 := 21656

	// Get first batch of delegator1 undelegations from contract
	batch0_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getUndelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetUndelegationsRet)
	test.unpack(batch0_parsed_result, "getUndelegations", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetUndelegationsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount, len(batch0_parsed_result.Undelegations))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[len(batch0_parsed_result.Undelegations)-1].address, batch0_parsed_result.Undelegations[len(batch0_parsed_result.Undelegations)-1].Validator)

	// Get second batch of delegator1 undelegations from contract
	batch1_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getUndelegations", delegator1_addr, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetUndelegationsRet)
	test.unpack(batch1_parsed_result, "getUndelegations", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetUndelegationsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount, len(batch1_parsed_result.Undelegations))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[dpos.GetUndelegationsMaxCount+len(batch1_parsed_result.Undelegations)-1].address, batch1_parsed_result.Undelegations[len(batch1_parsed_result.Undelegations)-1].Validator)

	// Get third batch of delegator1 undelegations from contract
	batch2_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getUndelegations", delegator1_addr, uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetUndelegationsRet)
	test.unpack(batch2_parsed_result, "getUndelegations", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetUndelegationsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount - 1
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount-1, len(batch2_parsed_result.Undelegations))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[2*dpos.GetUndelegationsMaxCount+len(batch2_parsed_result.Undelegations)-1].address, batch2_parsed_result.Undelegations[len(batch2_parsed_result.Undelegations)-1].Validator)

	// Get fourth batch of delegator1 undelegations from contract - it should return 0 undelegations
	batch3_result := test.ExecuteAndCheck(delegator1_addr, Big0, test.pack("getUndelegations", delegator1_addr, uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetUndelegationsRet)
	test.unpack(batch3_parsed_result, "getUndelegations", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned undelegations is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Undelegations))
	tc.Assert.Equal(true, batch3_parsed_result.End)
}

func TestGetValidator(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// ErrNonExistentValidator
	test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// Register validator and check if it is returned from contract
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	validator_raw := test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(DefaultMinimumDeposit, validator.ValidatorInfo.TotalStake)

	// Undelegate
	test.ExecuteAndCheck(val_owner, Big0, test.pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// Advance 3 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.AdvanceBlock(nil, nil, nil)
	test.ExecuteAndCheck(val_owner, Big0, test.pack("confirmUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrNonExistentValidator
	test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestGetTotalEligibleVotesCount(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	// Register validator and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// New delegation through registerValidator should not be applied yet in delayed storage - getTotalEligibleVotesCount should return 0 at this moment
	votes_count_raw := test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count := new(uint64)
	test.unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)
	tc.Assert.Equal(uint64(0), *votes_count)

	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count := bigutil.Div(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)

	// Delegate and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(delegator_addr, bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count = bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(3)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)

	// Undelegate and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(delegator_addr, Big0, test.pack("undelegate", val_addr, test.Chain_cfg.DPOS.EligibilityBalanceThreshold), util.ErrorString(""), util.ErrorString(""))
	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, Big0, test.pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count = bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)
}

func TestGetValidatorEligibleVotesCount(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	// Register validator
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Delegate some more
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// Wait DelegationDelay so new delegation is applied
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	// check if validator vote count was calculated properly in contract
	val_votes_count_raw := test.ExecuteAndCheck(val_addr, Big0, test.pack("getValidatorEligibleVotesCount", val_addr), util.ErrorString(""), util.ErrorString(""))
	val_votes_count := new(uint64)
	test.unpack(val_votes_count, "getValidatorEligibleVotesCount", val_votes_count_raw.CodeRetval)

	expected_votes_count := bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *val_votes_count)
}

func TestIsValidatorEligible(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	// Check if validatorEligible == false before register&delegate
	is_eligible_raw := test.ExecuteAndCheck(val_addr, Big0, test.pack("isValidatorEligible", val_addr), util.ErrorString(""), util.ErrorString(""))
	is_eligible := new(bool)
	test.unpack(is_eligible, "isValidatorEligible", is_eligible_raw.CodeRetval)
	tc.Assert.Equal(false, *is_eligible)

	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))

	// Wait DelegationDelay so new delegation is applied
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	// Check if validatorEligible == true after register&delegate
	is_eligible_raw = test.ExecuteAndCheck(val_addr, Big0, test.pack("isValidatorEligible", val_addr), util.ErrorString(""), util.ErrorString(""))
	is_eligible = new(bool)
	test.unpack(is_eligible, "isValidatorEligible", is_eligible_raw.CodeRetval)
	tc.Assert.Equal(true, *is_eligible)
}

func TestIterableMapClass(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.st.BeginBlock(&vm.BlockInfo{})

	var storage dpos.StorageWrapper
	evm_state := test.st.GetEvmState()
	storage.Init(dpos.EVMStateStorage{evm_state})

	iter_map_prefix := []byte{0}
	iter_map := dpos.IterableMap{}
	iter_map.Init(&storage, iter_map_prefix)

	acc1 := addr(1)
	acc2 := addr(2)
	acc3 := addr(3)
	acc4 := addr(4)

	// Tests CreateAccount & GetCount
	iter_map.CreateAccount(&acc1)
	tc.Assert.Equal(uint32(1), iter_map.GetCount())
	// Tries to create duplicate account
	tc.Assert.PanicsWithValue("Account already exists", func() { iter_map.CreateAccount(&acc1) })
	tc.Assert.Equal(uint32(1), iter_map.GetCount())

	iter_map.CreateAccount(&acc2)
	iter_map.CreateAccount(&acc3)
	iter_map.CreateAccount(&acc4)
	tc.Assert.Equal(uint32(4), iter_map.GetCount())

	// Tests GetAccounts
	items_in_batch := uint32(2)
	batch0_accounts, end := iter_map.GetAccounts(0 /* batch 0 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(batch0_accounts)))
	tc.Assert.Equal(acc1, batch0_accounts[0])
	tc.Assert.Equal(acc2, batch0_accounts[1])
	tc.Assert.Equal(false, end)

	batch1_accounts, end := iter_map.GetAccounts(1 /* batch 1 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(batch1_accounts)))
	tc.Assert.Equal(acc3, batch1_accounts[0])
	tc.Assert.Equal(acc4, batch1_accounts[1])
	tc.Assert.Equal(true, end)

	// Tests RemoveAccount
	iter_map.RemoveAccount(&acc2)
	tc.Assert.Equal(uint32(3), iter_map.GetCount())
	tc.Assert.PanicsWithValue("Account does not exist", func() { iter_map.RemoveAccount(&acc2) })
	tc.Assert.Equal(uint32(3), iter_map.GetCount())

	// To optimize iterbale map removing, it is implement through swapping of the item to be deleted with the last item
	// and then intenal array is just downsized by 1
	items_in_batch = uint32(3)
	accounts, end := iter_map.GetAccounts(0 /* batch 0 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(accounts)))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(acc1, accounts[0])
	// acc2 was deleted, so acc4 should be now at acc2 original position(index)
	tc.Assert.Equal(acc4, accounts[1])
	tc.Assert.Equal(acc3, accounts[2])

	// Tests AccountExists
	tc.Assert.Equal(true, iter_map.AccountExists(&acc1))
	tc.Assert.Equal(false, iter_map.AccountExists(&acc2))
	tc.Assert.Equal(true, iter_map.AccountExists(&acc3))
	tc.Assert.Equal(true, iter_map.AccountExists(&acc4))
}

func TestValidatorsClass(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.st.BeginBlock(&vm.BlockInfo{})

	var storage dpos.StorageWrapper
	evm_state := test.st.GetEvmState()
	storage.Init(dpos.EVMStateStorage{evm_state})

	validators := dpos.Validators{}
	field_validators := []byte{0}
	validators.Init(&storage, field_validators)

	validator1_addr, _ := generateAddrAndProof()
	validator1_owner := addr(1)

	validator2_addr, _ := generateAddrAndProof()
	validator2_owner := addr(1)

	// Checks CreateValidator & CheckValidatorOwner
	validators.CreateValidator(&validator1_owner, &validator1_addr, DefaultVrfKey, 0, 1, "validator1_description", "validator1_endpoint")
	validators.CheckValidatorOwner(&validator1_owner, &validator1_addr)
	tc.Assert.Equal(uint32(1), validators.GetValidatorsCount())

	validators.CreateValidator(&validator2_owner, &validator2_addr, DefaultVrfKey, 0, 2, "validator2_description", "validator2_endpoint")
	validators.CheckValidatorOwner(&validator2_owner, &validator2_addr)
	tc.Assert.Equal(uint32(2), validators.GetValidatorsCount())

	// Checks GetValidator & GetValidatorInfo
	validator1 := validators.GetValidator(&validator1_addr)
	tc.Assert.Equal(uint16(1), validator1.Commission)
	validator1_info := validators.GetValidatorInfo(&validator1_addr)
	tc.Assert.Equal("validator1_description", validator1_info.Description)
	tc.Assert.Equal("validator1_endpoint", validator1_info.Endpoint)

	// Checks ModifyValidator & ModifyValidatorInfo
	validator1.Commission = 11
	validator1_info.Description = "validator1_description_modified"
	validator1_info.Endpoint = "validator1_endpoint_modified"
	validators.ModifyValidator(&validator1_addr, validator1)
	validators.ModifyValidatorInfo(&validator1_addr, validator1_info)

	validator1 = validators.GetValidator(&validator1_addr)
	tc.Assert.Equal(uint16(11), validator1.Commission)
	validator1_info = validators.GetValidatorInfo(&validator1_addr)
	tc.Assert.Equal("validator1_description_modified", validator1_info.Description)
	tc.Assert.Equal("validator1_endpoint_modified", validator1_info.Endpoint)

	// Checks GetValidatorsAddresses
	validators_addresses, end := validators.GetValidatorsAddresses(uint32(0), uint32(2))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(2, len(validators_addresses))
	tc.Assert.Equal(validators.GetValidatorsCount(), uint32(len(validators_addresses)))
	tc.Assert.Equal(validator1_addr, validators_addresses[0])
	tc.Assert.Equal(validator2_addr, validators_addresses[1])

	// Checks DeleteValidator
	tc.Assert.Equal(true, validators.ValidatorExists(&validator1_addr))
	validators.DeleteValidator(&validator1_addr)
	tc.Assert.Equal(false, validators.ValidatorExists(&validator1_addr))
	tc.Assert.Equal(uint32(1), validators.GetValidatorsCount())

	validator3_addr := addr(3)
	tc.Assert.PanicsWithValue("ModifyValidator: non existent validator", func() { validators.ModifyValidator(&validator3_addr, validator1) })
	tc.Assert.PanicsWithValue("ModifyValidatorInfo: non existent validator", func() { validators.ModifyValidatorInfo(&validator3_addr, validator1_info) })
}

func TestDelegationsClass(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.st.BeginBlock(&vm.BlockInfo{})

	var storage dpos.StorageWrapper
	evm_state := test.st.GetEvmState()
	storage.Init(dpos.EVMStateStorage{evm_state})

	delegations := dpos.Delegations{}
	field_delegations := []byte{2}
	delegations.Init(&storage, field_delegations)

	validator1_addr := addr(1)
	validator2_addr := addr(2)

	delegator1_addr := addr(3)

	// Check getters to 0 values
	tc.Assert.Equal(false, delegations.DelegationExists(&delegator1_addr, &validator1_addr))
	tc.Assert.Equal(uint32(0), delegations.GetDelegationsCount(&delegator1_addr))

	delegations_ret, end := delegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(0, len(delegations_ret))
	tc.Assert.Equal(true, end)

	delegation_ret := delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	var delegation_nil_ptr *dpos.Delegation = nil
	tc.Assert.Equal(delegation_nil_ptr, delegation_ret)

	// Creates 2 delegations
	delegations.CreateDelegation(&delegator1_addr, &validator1_addr, 0, Big50)
	delegations.CreateDelegation(&delegator1_addr, &validator2_addr, 0, Big50)

	// Check GetDelegationsCount + DelegationExists
	tc.Assert.Equal(uint32(2), delegations.GetDelegationsCount(&delegator1_addr))
	tc.Assert.Equal(true, delegations.DelegationExists(&delegator1_addr, &validator1_addr))

	// Check GetDelegatorValidatorsAddresses
	delegations_ret, end = delegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(2, len(delegations_ret))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(validator1_addr, delegations_ret[0])
	tc.Assert.Equal(validator2_addr, delegations_ret[1])

	// Check GetDelegation
	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(0), delegation_ret.LastUpdated)
	tc.Assert.Equal(Big50, delegation_ret.Stake)

	// Check ModifyDelegation
	delegation_ret.LastUpdated = 1
	delegation_ret.Stake = Big10
	delegations.ModifyDelegation(&delegator1_addr, &validator1_addr, delegation_ret)

	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(1), delegation_ret.LastUpdated)
	tc.Assert.Equal(Big10, delegation_ret.Stake)

	// Check RemoveDelegation
	delegations.RemoveDelegation(&delegator1_addr, &validator1_addr)
	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(delegation_nil_ptr, delegation_ret)
	tc.Assert.Equal(uint32(1), delegations.GetDelegationsCount(&delegator1_addr))
	tc.Assert.Equal(false, delegations.DelegationExists(&delegator1_addr, &validator1_addr))
}

func TestUndelegationsClass(t *testing.T) {
	tc, test := init_test(t, CopyDefaulChainConfig())
	defer test.end()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.st.BeginBlock(&vm.BlockInfo{})

	var storage dpos.StorageWrapper
	evm_state := test.st.GetEvmState()
	storage.Init(dpos.EVMStateStorage{evm_state})

	undelegations := dpos.Undelegations{}
	field_undelegations := []byte{3}
	undelegations.Init(&storage, field_undelegations)

	validator1_addr := addr(1)
	validator2_addr := addr(2)

	delegator1_addr := addr(3)

	// Check getters to 0 values
	tc.Assert.Equal(false, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr))
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsCount(&delegator1_addr))

	undelegations_ret, end := undelegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(0, len(undelegations_ret))
	tc.Assert.Equal(true, end)

	undelegation_ret := undelegations.GetUndelegation(&delegator1_addr, &validator1_addr)
	var undelegation_nil_ptr *dpos.Undelegation = nil
	tc.Assert.Equal(undelegation_nil_ptr, undelegation_ret)

	// Creates 2 delegations
	undelegations.CreateUndelegation(&delegator1_addr, &validator1_addr, 0, Big50)
	undelegations.CreateUndelegation(&delegator1_addr, &validator2_addr, 0, Big50)

	// Check GetUndelegationsCount + UndelegationExists
	tc.Assert.Equal(uint32(2), undelegations.GetUndelegationsCount(&delegator1_addr))
	tc.Assert.Equal(true, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr))

	// Check GetUndelegations
	undelegations_ret, end = undelegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(2, len(undelegations_ret))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(validator1_addr, undelegations_ret[0])
	tc.Assert.Equal(validator2_addr, undelegations_ret[1])

	// Check GetUndelegation
	undelegation_ret = undelegations.GetUndelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(0), undelegation_ret.Block)
	tc.Assert.Equal(Big50, undelegation_ret.Amount)

	// Check RemoveDelegation
	undelegations.RemoveUndelegation(&delegator1_addr, &validator1_addr)
	undelegation_ret = undelegations.GetUndelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(undelegation_nil_ptr, undelegation_ret)
	tc.Assert.Equal(uint32(1), undelegations.GetUndelegationsCount(&delegator1_addr))
	tc.Assert.Equal(false, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr))
}
