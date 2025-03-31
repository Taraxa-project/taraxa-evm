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
	DefaultBalance                     = bigutil.Mul(big.NewInt(2050000000), TaraPrecision)
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
			AspenHf: chain_config.AspenHfConfig{
				BlockNumPartOne: 0,
				BlockNumPartTwo: 0,
				// Max token supply is 12 Billion TARA -> 12e+9(12 billion) * 1e+18(tara precision)
				MaxSupply:        new(big.Int).Mul(big.NewInt(12e+9), big.NewInt(1e+18)),
				GeneratedRewards: big.NewInt(0),
			},
			CornusHf: chain_config.CornusHfConfig{
				BlockNum:                0,
				DelegationLockingPeriod: 4,
				DagGasLimit:             100000,
				PbftGasLimit:            1000000,
			},
			SoleiroliaHf: chain_config.SoleiroliaHfConfig{
				BlockNum:       0,
				TrxMinGasPrice: 1,
				TrxMaxGasLimit: 1,
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

func addValidator(cfg *chain_config.ChainConfig) (privkey []byte, address common.Address) {
	pubkey, privkey := generateKeyPair()
	address = common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
	cfg.DPOS.InitialValidators = append(cfg.DPOS.InitialValidators, chain_config.GenesisValidator{Address: address, Owner: address, VrfKey: common.Hash{}.Bytes(), Commission: 0, Endpoint: "", Description: "", Delegations: map[common.Address]*big.Int{addr(1): DefaultValidatorMaximumStake}})
	return
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
	cfg := DefaultChainCfg
	privkey, _ := addValidator(&cfg)
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, cfg)
	defer test.End()

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
	cfg := DefaultChainCfg
	privkey, _ := addValidator(&cfg)
	_, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, cfg)
	defer test.End()

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
	cfg := DefaultChainCfg
	privkey1, malicious_vote_author1 := addValidator(&cfg)
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, cfg)
	defer test.End()

	proof_author := addr(1)

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
	tc.Assert.Equal(1+2*uint64(test.Chain_cfg.DPOS.DelegationDelay)+DefaultChainCfg.Hardforks.MagnoliaHf.JailTime, *result_parsed)
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

func TestJailedValidatorsList(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.MagnoliaHf.JailTime = 50
	proof_author := addr(1)
	test_voters_count := uint64(10)

	type VoteProof struct {
		VoteA  slashing.Vote
		VoteB  slashing.Vote
		author common.Address
	}

	proofs := make([]VoteProof, test_voters_count)
	// execute commitDoubleVotingProof + generate DelegationDelay blocks
	for i := uint64(0); i < test_voters_count; i++ {
		privkey1, author := addValidator(&cfg)
		proofs[i].author = author
		proofs[i].VoteA = DefaultVote
		proofs[i].VoteA.VrfSortition.Period = uint64(i)
		signVote(&proofs[i].VoteA, privkey1)

		voteB := proofs[i].VoteA
		voteB.BlockHash = common.Hash{0x2}
		signVote(&voteB, privkey1)
		proofs[i].VoteB = voteB
	}

	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, cfg)
	defer test.End()

	for _, proof := range proofs {
		test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&proof.VoteA), GetVoteRlp(&proof.VoteB)), util.ErrorString(""), util.ErrorString(""))
		for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
			test.AdvanceBlock(nil, nil)
		}
	}

	// Check list of jailed validators. We need to get it directly from reader later
	jailed_from_contract := test.GetJailedValidators()
	tc.Assert.Equal(test_voters_count, uint64(len(jailed_from_contract)))
	{
		jailed := make([]common.Address, test_voters_count)
		for i := uint64(0); i < test_voters_count; i++ {
			jailed[i] = proofs[i].author
		}
		tc.Assert.Equal(jailed, jailed_from_contract)
	}
	first_unjail_block := uint64(0)
	// Every execute is + 1 block, so add it to delegation delay
	blocks_per_iteration := uint64(test.Chain_cfg.DPOS.DelegationDelay + 1)
	for i := uint64(0); i < test_voters_count; i++ {
		result := test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("getJailBlock", proofs[i].author), util.ErrorString(""), util.ErrorString(""))
		unjail_block := uint64(0)
		test.Unpack(&unjail_block, "getJailBlock", result.CodeRetval)
		expected_unjail := 1 + cfg.Hardforks.MagnoliaHf.JailTime + i*blocks_per_iteration
		tc.Assert.Equal(int(expected_unjail), int(unjail_block))

		if first_unjail_block == 0 {
			first_unjail_block = unjail_block
		}
	}

	if first_unjail_block == 0 {
		return
	}
	for {
		test.AdvanceBlock(nil, nil)
		if test.BlockNumber() == first_unjail_block {
			break
		}
	}

	for i := uint64(0); i < test_voters_count; i++ {
		jailed_from_contract := test.GetJailedValidators()
		tc.Assert.Equal(test_voters_count-i, uint64(len(jailed_from_contract)))
		jailed := make([]common.Address, test_voters_count)
		for i := uint64(0); i < test_voters_count; i++ {
			jailed[i] = proofs[i].author
		}
		tc.Assert.Equal(jailed[i:], jailed_from_contract)

		// Every get is + 1 block, so advance only by DelegationDelay
		for i := uint64(0); i < uint64(test.Chain_cfg.DPOS.DelegationDelay); i++ {
			test.AdvanceBlock(nil, nil)
		}
	}
}

func TestDoubleJailing(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.MagnoliaHf.JailTime = 50
	privkey1, _ := addValidator(&cfg)
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, cfg)
	defer test.End()

	proof_author := addr(1)

	vote_a := DefaultVote
	vote_a.VrfSortition.Period = uint64(1)
	signVote(&vote_a, privkey1)

	vote_b := vote_a
	vote_b.BlockHash = common.Hash{0x2}
	signVote(&vote_b, privkey1)

	vote_c := vote_a
	vote_c.BlockHash = common.Hash{0x3}
	signVote(&vote_c, privkey1)

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_b)), util.ErrorString(""), util.ErrorString(""))
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", GetVoteRlp(&vote_a), GetVoteRlp(&vote_c)), util.ErrorString(""), util.ErrorString(""))
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	jailed_from_contract := test.GetJailedValidators()
	tc.Assert.Equal(1, len(jailed_from_contract))

}
