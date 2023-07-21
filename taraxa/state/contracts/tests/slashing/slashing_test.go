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

func TestCommitDoubleVotingProof(t *testing.T) {
	tc, test := test_utils.Init_test(slashing.ContractAddress(), slashing_sol.TaraxaSlashingClientMetaData, t, DefaultChainCfg)
	defer test.End()

	pubkey, seckey := generateKeyPair()
	malicious_vote_author := common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])

	var vote slashing.Vote
	vote.BlockHash = common.Hash{0x1}
	vote.VrfSortition.Period = 10
	vote.VrfSortition.Round = 20
	vote.VrfSortition.Step = 30
	vote.VrfSortition.Proof = [80]byte{1, 2, 3}

	// Sign vote
	sig, err := secp256k1.Sign(vote.GetHash().Bytes(), seckey)
	tc.Assert.True(err == nil)

	copy(vote.Signature[:], sig)

	proof_author := addr(1)
	test.ExecuteAndCheck(proof_author, big.NewInt(0), test.Pack("commitDoubleVotingProof", proof_author, malicious_vote_author, vote.GetRlp(true), vote.GetRlp(true)), util.ErrorString(""), util.ErrorString(""))
}
