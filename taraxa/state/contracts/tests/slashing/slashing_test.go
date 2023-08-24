package slashing_tests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"strings"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto/secp256k1"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	slashing_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	test_utils "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/tests"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"
)

// This strings should correspond to event signatures in ../solidity/slashing_contract_interface.sol file
var JailedEventHash = *keccak256.Hash([]byte("Jailed(address,uint64,uint64,uint8)"))

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
		DPOS: chain_config.DPOSConfig{
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
		Hardforks: chain_config.HardforksConfig{
			FixRedelegateBlockNum: 0,
			MagnoliaHf: chain_config.MagnoliaHfConfig{
				BlockNum: 0,
				JailTime: 5,
			},
		},
	}
)

var DefaultVote = slashing.Vote{BlockHash: common.Hash{0x1}, VrfSortition: slashing.VrfPbftSortition{Period: 10, Round: 20, Step: 30, Proof: [80]byte{1, 2, 3}}}

func signVote(vote *slashing.Vote, private_key []byte) {
	vote.VrfSortitionBytes = rlp.MustEncodeToBytes(vote.VrfSortition)

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

func GetVoteRlp(vote *slashing.Vote) []byte {
	return rlp.MustEncodeToBytes(vote)
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
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote), GetVoteRlp(&vote)), slashing.ErrIdenticalVotes, util.ErrorString(""))
}

func TestDoubleVotingExistingProof(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote_a := DefaultVote
	signVote(&vote_a, privkey)

	vote_b := DefaultVote
	vote_b.BlockHash = common.Hash{0x2}
	signVote(&vote_b, privkey)

	proof_author := addr(1)
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), util.ErrorString(""), util.ErrorString(""))
	// Existing proof err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrExistingDoubleVotingProof, util.ErrorString(""))

	// Existing proof err - change votes order
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_b), GetVoteRlp(&vote_a)), slashing.ErrExistingDoubleVotingProof, util.ErrorString(""))
}

func TestDoubleVotingInvalidSig(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote_a := DefaultVote
	vote_a.VrfSortitionBytes = rlp.MustEncodeToBytes(vote_a.VrfSortition)

	vote_b := DefaultVote
	vote_b.BlockHash = common.Hash{0x2}
	vote_b.VrfSortitionBytes = rlp.MustEncodeToBytes(vote_b.VrfSortition)

	proof_author := addr(1)
	// Invalid signature err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVoteSignature, util.ErrorString(""))

	vote_a = DefaultVote
	signVote(&vote_a, privkey)
	// Change vote data after it was signed
	vote_a.BlockHash = common.Hash{0x9}
	vote_a.VrfSortitionBytes = rlp.MustEncodeToBytes(vote_a.VrfSortition)

	vote_b = DefaultVote
	vote_b.BlockHash = common.Hash{0x2}
	signVote(&vote_b, privkey)

	// Wrong address recovered - vote_a data changes after signing it
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesValidator, util.ErrorString(""))
}

func TestDoubleVotingInvalidPeriodRoundStep(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote_a := DefaultVote
	signVote(&vote_a, privkey)

	vote_b := DefaultVote
	vote_b.VrfSortition.Period = vote_a.VrfSortition.Period + 1
	signVote(&vote_b, privkey)

	proof_author := addr(1)
	// Existing invalid period
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))

	vote_a = DefaultVote
	vote_a.VrfSortition.Round = 10
	signVote(&vote_a, privkey)

	vote_b = DefaultVote
	vote_b.VrfSortition.Round = vote_a.VrfSortition.Round + 1
	signVote(&vote_b, privkey)
	// Existing invalid round
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))

	vote_a = DefaultVote
	vote_a.VrfSortition.Step = 10
	signVote(&vote_a, privkey)

	vote_b = DefaultVote
	vote_b.VrfSortition.Step = vote_a.VrfSortition.Step + 1
	signVote(&vote_b, privkey)
	// Existing invalid step
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesPeriodRoundStep, util.ErrorString(""))
}

func TestDoubleVotingInvalidBlockHash(t *testing.T) {
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	_, privkey := generateKeyPair()
	vote_a := DefaultVote
	signVote(&vote_a, privkey)

	vote_b := DefaultVote
	vote_b.VrfSortition.Proof = [80]byte{4, 5, 6}
	signVote(&vote_b, privkey)

	proof_author := addr(1)
	// Invalid block hash err
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))

	vote_a = DefaultVote
	vote_a.VrfSortition.Step = 5
	vote_b.BlockHash = common.Hash{0x1}
	signVote(&vote_a, privkey)

	vote_b = DefaultVote
	vote_b.VrfSortition.Step = 5
	vote_b.BlockHash = common.ZeroHash
	signVote(&vote_b, privkey)

	// Invalid block hash err - second finish step, 1 specific block + 1 null block hash is allowed
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))

	vote_a = DefaultVote
	vote_a.VrfSortition.Step = 5
	vote_a.BlockHash = common.ZeroHash
	signVote(&vote_a, privkey)

	vote_b = DefaultVote
	vote_b.VrfSortition.Step = 5
	vote_b.BlockHash = common.Hash{0x1}
	signVote(&vote_b, privkey)

	// Invalid block hash err - second finish step, 1 specific block + 1 null block hash is allowed
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), slashing.ErrInvalidVotesBlockHash, util.ErrorString(""))
}

func TestGetJailBlock(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	proof_author := addr(1)

	pubkey1, privkey1 := generateKeyPair()
	malicious_vote_author1 := common.BytesToAddress(keccak256.Hash(pubkey1[1:])[12:])
	vote_a := DefaultVote
	signVote(&vote_a, privkey1)

	vote_b := DefaultVote
	vote_b.BlockHash = common.Hash{0x2}
	signVote(&vote_b, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), util.ErrorString(""), util.ErrorString(""))

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailBlock", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed := new(uint64)
	test.Unpack(result_parsed, "getJailBlock", result.CodeRetval)
	tc.Assert.Equal(1+DefaultChainCfg.Hardforks.MagnoliaHf.JailTime, *result_parsed)

	// Test cumulative jail time - commit another double voting proof
	vote_a = DefaultVote
	signVote(&vote_a, privkey1)

	vote_b = DefaultVote
	vote_b.BlockHash = common.Hash{0x3}
	signVote(&vote_b, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), util.ErrorString(""), util.ErrorString(""))

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	result = test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailBlock", malicious_vote_author1), util.ErrorString(""), util.ErrorString(""))
	result_parsed = new(uint64)
	test.Unpack(result_parsed, "getJailBlock", result.CodeRetval)
	tc.Assert.Equal(1+2*DefaultChainCfg.Hardforks.MagnoliaHf.JailTime, *result_parsed)
}

func TestMakeLogsCheckTopics(t *testing.T) {
	tc := tests.NewTestCtx(t)
	block := uint64(123)

	Abi, _ := abi.JSON(strings.NewReader(slashing_sol.TaraxaSlashingClientMetaData))
	logs := *new(slashing.Logs).Init(Abi.Events)

	count := 0
	{
		log := logs.MakeJailedLog(&common.ZeroAddress, block, block+DefaultChainCfg.Hardforks.MagnoliaHf.JailTime, slashing.DOUBLE_VOTING)
		tc.Assert.Equal(log.Topics[0], JailedEventHash)
		count++
	}

	// Check that we tested all events from the ABI
	tc.Assert.Equal(count, len(Abi.Events))
}
