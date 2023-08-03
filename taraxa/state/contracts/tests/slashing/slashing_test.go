package slashing_tests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto/secp256k1"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	slashing_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	test_utils "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/tests"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"
)

type IsJailedRet struct {
	End bool
}

type GenesisBalances = map[common.Address]*big.Int

var addr, addr_p = tests.Addr, tests.AddrP

var (
	TaraPrecision                      = big.NewInt(1e+18)
	DefaultBalance                     = bigutil.Mul(big.NewInt(5000000), TaraPrecision)
	DefaultEligibilityBalanceThreshold = bigutil.Mul(big.NewInt(1000000), TaraPrecision)
	DefaultVoteEligibilityBalanceStep  = bigutil.Mul(big.NewInt(1000), TaraPrecision)
	DefaultValidatorMaximumStake       = bigutil.Mul(big.NewInt(10000000), TaraPrecision)
	DefaultMinimumDeposit              = bigutil.Mul(big.NewInt(1000), TaraPrecision)
	DefaultVrfKey                      = common.RightPadBytes([]byte("0x0"), 32)

	DefaultChainCfg = chain_config.ChainConfig{
		GenesisBalances: GenesisBalances{addr(1): DefaultBalance, addr(2): DefaultBalance, addr(3): DefaultBalance, addr(4): DefaultBalance, addr(5): DefaultBalance},
		DPOS: dpos.Config{
			EligibilityBalanceThreshold: DefaultEligibilityBalanceThreshold,
			VoteEligibilityBalanceStep:  DefaultVoteEligibilityBalanceStep,
			ValidatorMaximumStake:       DefaultValidatorMaximumStake,
			MinimumDeposit:              DefaultMinimumDeposit,
			MaxBlockAuthorReward:        10,
			DagProposersReward:          50,
			CommissionChangeDelta:       0,
			CommissionChangeFrequency:   0,
			DelegationDelay:             2,
			DelegationLockingPeriod:     4,
			BlocksPerYear:               365 * 24 * 60 * 15, // block every 4 seconds
			YieldPercentage:             20,
		},
		Slashing: slashing.Config{5},
	}
)

var DefaultVote = slashing.Vote{BlockHash: common.Hash{0x1}, VrfSortition: slashing.VrfPbftSortition{Period: 10, Round: 20, Step: 30, Proof: [80]byte{1, 2, 3}}}

func signVote(vote *slashing.Vote, private_key []byte) {
	// Sign vote
	sig, err := secp256k1.Sign(vote.GetHash().Bytes(), private_key)
	if err != nil {
		panic("Unable to sign vote")
	}

	copy(vote.Signature[:], sig)
}

func generateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)

	return pubkey, privkey
}

func TestDoubleVotingSameVotesHashes(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote := DefaultVote
	signVote(&vote, privkey)

	// Sign vote
	sig, err := secp256k1.Sign(vote.GetHash().Bytes(), privkey)
	tc.Assert.True(err == nil)

	copy(vote.Signature[:], sig)

	proof_author := addr(1)
	// Same vote hash err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote.GetRlp(true), vote.GetRlp(true)), slashing.ErrIdenticalVotes, util.ErrorString(""))
}

func TestDoubleVotingExistingProof(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote1 := DefaultVote
	signVote(&vote1, privkey)

	vote2 := DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey)

	proof_author := addr(1)
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))
	// Existing proof err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrExistingDoubleVotingProof, util.ErrorString(""))

	// Existing proof err - change votes order
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote2.GetRlp(true), vote1.GetRlp(true)), slashing.ErrExistingDoubleVotingProof, util.ErrorString(""))
}

func TestDoubleVotingInvalidSig(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote1 := DefaultVote

	vote2 := DefaultVote
	vote2.BlockHash = common.Hash{0x2}

	proof_author := addr(1)
	// Invalid signature err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVoteSignature, util.ErrorString(""))

	vote1 = DefaultVote
	signVote(&vote1, privkey)
	vote1.VrfSortition.Period = 123

	vote2 = DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey)

	// Invalid signature err - vote1 data changes after signing it
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesValidator, util.ErrorString(""))
}

func TestDoubleVotingInvalidPeriodRoundStep(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote1 := DefaultVote
	signVote(&vote1, privkey)

	vote2 := DefaultVote
	vote2.VrfSortition.Period = vote1.VrfSortition.Period + 1
	signVote(&vote2, privkey)

	proof_author := addr(1)
	// Existing invalid period
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))

	vote1 = DefaultVote
	vote1.VrfSortition.Round = 10
	signVote(&vote1, privkey)

	vote2 = DefaultVote
	vote2.VrfSortition.Round = vote1.VrfSortition.Round + 1
	signVote(&vote2, privkey)
	// Existing invalid round
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))

	vote1 = DefaultVote
	vote1.VrfSortition.Step = 10
	signVote(&vote1, privkey)

	vote2 = DefaultVote
	vote2.VrfSortition.Step = vote1.VrfSortition.Step + 1
	signVote(&vote2, privkey)
	// Existing invalid step
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))
}

func TestDoubleVotingInvalidBlockHash(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote1 := DefaultVote
	signVote(&vote1, privkey)

	vote2 := DefaultVote
	vote2.VrfSortition.Proof = [80]byte{4, 5, 6}
	signVote(&vote2, privkey)

	proof_author := addr(1)
	// Invalid block hash err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))

	vote1 = DefaultVote
	vote1.VrfSortition.Step = 5
	vote2.BlockHash = common.Hash{0x1}
	signVote(&vote1, privkey)

	vote2 = DefaultVote
	vote2.VrfSortition.Step = 5
	vote2.BlockHash = common.ZeroHash
	signVote(&vote2, privkey)

	// Invalid block hash err - second finish step, 1 specific block + 1 null block hash is allowed
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))

	vote1 = DefaultVote
	vote1.VrfSortition.Step = 5
	vote1.BlockHash = common.ZeroHash
	signVote(&vote1, privkey)

	vote2 = DefaultVote
	vote2.VrfSortition.Step = 5
	vote2.BlockHash = common.Hash{0x1}
	signVote(&vote2, privkey)

	// Invalid block hash err - second finish step, 1 specific block + 1 null block hash is allowed
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))
}

func TestIsJailed(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	proof_author := addr(1)

	pubkey1, privkey1 := generateKeyPair()
	malicious_vote_author1 := common.BytesToAddress(keccak256.Hash(pubkey1[1:])[12:])
	vote1 := DefaultVote
	signVote(&vote1, privkey1)

	vote2 := DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))

	result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("isJailed", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed := new(bool)
	test.Unpack(result_parsed, "isJailed", result.CodeRetval)
	tc.Assert.Equal(true, *result_parsed)

	// Advance couple of blocks and check if IsJailed flag is set to false
	for idx := uint64(0); idx < DefaultChainCfg.Slashing.DoubleVotingJailTime; idx++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	result = test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("isJailed", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed = new(bool)
	test.Unpack(result_parsed, "isJailed", result.CodeRetval)
	tc.Assert.Equal(false, *result_parsed)
}

func TestGetJailInfo(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	proof_author := addr(1)

	pubkey1, privkey1 := generateKeyPair()
	malicious_vote_author1 := common.BytesToAddress(keccak256.Hash(pubkey1[1:])[12:])
	vote1 := DefaultVote
	signVote(&vote1, privkey1)

	vote2 := DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))

	type GetJailInfoRet struct {
		Info slashing_sol.SlashingInterfaceJailInfo
	}

	result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailInfo", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed := new(GetJailInfoRet)
	test.Unpack(result_parsed, "getJailInfo", result.CodeRetval)

	tc.Assert.Equal(true, result_parsed.Info.IsJailed)
	tc.Assert.Equal(uint32(1), result_parsed.Info.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(1), big.NewInt(int64(DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed.Info.JailBlock)

	// Test cumulative jail time - commit another double voting proof
	vote1 = DefaultVote
	signVote(&vote1, privkey1)

	vote2 = DefaultVote
	vote2.BlockHash = common.Hash{0x3}
	signVote(&vote2, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))

	result = test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailInfo", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed = new(GetJailInfoRet)
	test.Unpack(result_parsed, "getJailInfo", result.CodeRetval)

	tc.Assert.Equal(true, result_parsed.Info.IsJailed)
	tc.Assert.Equal(uint32(2), result_parsed.Info.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(1), big.NewInt(int64(2*DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed.Info.JailBlock)

	// Advance couple of blocks and check if IsJailed flag is set to false
	for idx := uint64(0); idx < 2*DefaultChainCfg.Slashing.DoubleVotingJailTime; idx++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	result = test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailInfo", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed = new(GetJailInfoRet)
	test.Unpack(result_parsed, "getJailInfo", result.CodeRetval)

	tc.Assert.Equal(false, result_parsed.Info.IsJailed)
	tc.Assert.Equal(uint32(2), result_parsed.Info.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(1), big.NewInt(int64(2*DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed.Info.JailBlock)
}

func TestMaliciousValidatorsList(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	proof_author := addr(1)

	pubkey1, privkey1 := generateKeyPair()
	malicious_vote_author1 := common.BytesToAddress(keccak256.Hash(pubkey1[1:])[12:])
	vote1 := DefaultVote
	signVote(&vote1, privkey1)

	vote2 := DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))

	pubkey2, privkey2 := generateKeyPair()
	malicious_vote_author2 := common.BytesToAddress(keccak256.Hash(pubkey2[1:])[12:])
	vote1 = DefaultVote
	signVote(&vote1, privkey2)

	vote2 = DefaultVote
	vote2.BlockHash = common.Hash{0x2}
	signVote(&vote2, privkey2)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote2.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))

	result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getMaliciousValidators"), util.ErrorString(""), util.ErrorString(""))
	result_parsed := []slashing_sol.SlashingInterfaceMaliciousValidator{}
	test.Unpack(&result_parsed, "getMaliciousValidators", result.CodeRetval)
	tc.Assert.Equal(2, len(result_parsed))

	tc.Assert.Equal(malicious_vote_author1.Bytes(), result_parsed[0].Validator.Bytes())
	tc.Assert.Equal(true, result_parsed[0].JailInfo.IsJailed)
	tc.Assert.Equal(uint32(1), result_parsed[0].JailInfo.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(1), big.NewInt(int64(DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed[0].JailInfo.JailBlock)

	tc.Assert.Equal(malicious_vote_author2.Bytes(), result_parsed[1].Validator.Bytes())
	tc.Assert.Equal(true, result_parsed[1].JailInfo.IsJailed)
	tc.Assert.Equal(uint32(1), result_parsed[1].JailInfo.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(2), big.NewInt(int64(DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed[1].JailInfo.JailBlock)

	// Advance couple of blocks and check if IsJailed flag is set to false
	for idx := uint64(0); idx < DefaultChainCfg.Slashing.DoubleVotingJailTime; idx++ {
		test.AdvanceBlock(nil, nil, nil)
	}

	result = test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getMaliciousValidators"), util.ErrorString(""), util.ErrorString(""))
	result_parsed = []slashing_sol.SlashingInterfaceMaliciousValidator{}
	test.Unpack(&result_parsed, "getMaliciousValidators", result.CodeRetval)
	tc.Assert.Equal(2, len(result_parsed))

	tc.Assert.Equal(malicious_vote_author1.Bytes(), result_parsed[0].Validator.Bytes())
	tc.Assert.Equal(false, result_parsed[0].JailInfo.IsJailed)
	tc.Assert.Equal(uint32(1), result_parsed[0].JailInfo.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(1), big.NewInt(int64(DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed[0].JailInfo.JailBlock)

	tc.Assert.Equal(malicious_vote_author2.Bytes(), result_parsed[1].Validator.Bytes())
	tc.Assert.Equal(false, result_parsed[1].JailInfo.IsJailed)
	tc.Assert.Equal(uint32(1), result_parsed[1].JailInfo.ProofsCount)
	tc.Assert.Equal(bigutil.Add(big.NewInt(2), big.NewInt(int64(DefaultChainCfg.Slashing.DoubleVotingJailTime))), result_parsed[1].JailInfo.JailBlock)
}

func TestDoubleVotingProofsList(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	proof_author := addr(1)
	pubkey1, privkey1 := generateKeyPair()
	malicious_vote_author1 := common.BytesToAddress(keccak256.Hash(pubkey1[1:])[12:])

	// Commit 3 double voting proofs
	vote1 := DefaultVote
	signVote(&vote1, privkey1)

	votes_count := 3
	votes := make([]slashing.Vote, votes_count)

	for idx := 0; idx < votes_count; idx++ {
		hash_bytes := make([]byte, 1)
		hash_bytes[0] = byte(idx)

		vote := DefaultVote
		vote.BlockHash = common.BytesToHash(hash_bytes)
		signVote(&vote, privkey1)
		votes[idx] = vote

		test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", vote1.GetRlp(true), vote.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))
	}

	result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getDoubleVotingProofs", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	var result_parsed []slashing_sol.SlashingInterfaceDoubleVotingProof
	test.Unpack(&result_parsed, "getDoubleVotingProofs", result.CodeRetval)
	tc.Assert.Equal(3, len(result_parsed))

	for idx := 0; idx < votes_count; idx++ {
		tc.Assert.Equal(proof_author.Bytes(), result_parsed[idx].ProofAuthor.Bytes())
		tc.Assert.Equal(big.NewInt(int64(idx)+1), result_parsed[idx].Block)
	}
}
