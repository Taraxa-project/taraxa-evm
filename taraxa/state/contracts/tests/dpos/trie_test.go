package dpos_tests

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
)

type ContractTest struct {
	ChainCfg     chain_config.ChainConfig
	st           state.StateTransition
	statedb      *state_db_rocksdb.DB
	tc           *tests.TestCtx
	SUT          *state.API
	blk_n        types.BlockNum
	abi          abi.ABI
	ContractAddr *common.Address
	Sender       common.Address
}

var (
	BigZero = big.NewInt(0)
)

func init_contract_test(t *testing.T, cfg chain_config.ChainConfig) (tc tests.TestCtx, test ContractTest) {
	tc = tests.NewTestCtx(t)
	test.init(&tc, cfg)
	return
}

func (self *ContractTest) init(t *tests.TestCtx, cfg chain_config.ChainConfig) {
	self.tc = t
	self.ChainCfg = cfg

	self.Sender = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
	self.ChainCfg.GenesisBalances[self.Sender] = big.NewInt(int64(9000000000000000000))

	self.statedb = new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: self.tc.DataDir(),
	})
	self.SUT = new(state.API).Init(
		self.statedb,
		func(num types.BlockNum) *big.Int { panic("unexpected") },
		&self.ChainCfg,
		state.APIOpts{},
	)

	self.st = self.SUT.GetStateTransition()
	simpleStorageCode := common.Hex2Bytes("608060405234801561001057600080fd5b506000606490505b60fa811015610069578060008190555060648161003591906100a8565b6001819055506001546002600080548152602001908152602001600020819055508080610061906100dc565b915050610018565b50610124565b6000819050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60006100b38261006f565b91506100be8361006f565b92508282019050808211156100d6576100d5610079565b5b92915050565b60006100e78261006f565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361011957610118610079565b5b600182019050919050565b61031c806101336000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80631ab06ee51461005157806329e99f071461006d5780636d4ce63c1461009d578063fbf611a7146100bc575b600080fd5b61006b600480360381019061006691906101ad565b6100ec565b005b610087600480360381019061008291906101ed565b610116565b6040516100949190610229565b60405180910390f35b6100a561012e565b6040516100b3929190610244565b60405180910390f35b6100d660048036038101906100d191906101ed565b61013f565b6040516100e39190610286565b60405180910390f35b81600081905550806001819055508060026000848152602001908152602001600020819055505050565b60026020528060005260406000206000915090505481565b600080600054600154915091509091565b60008160026040516020016101559291906102bd565b604051602081830303815290604052805190602001209050919050565b600080fd5b6000819050919050565b61018a81610177565b811461019557600080fd5b50565b6000813590506101a781610181565b92915050565b600080604083850312156101c4576101c3610172565b5b60006101d285828601610198565b92505060206101e385828601610198565b9150509250929050565b60006020828403121561020357610202610172565b5b600061021184828501610198565b91505092915050565b61022381610177565b82525050565b600060208201905061023e600083018461021a565b92915050565b6000604082019050610259600083018561021a565b610266602083018461021a565b9392505050565b6000819050919050565b6102808161026d565b82525050565b600060208201905061029b6000830184610277565b92915050565b600060ff82169050919050565b6102b7816102a1565b82525050565b60006040820190506102d2600083018561021a565b6102df60208301846102ae565b939250505056fea2646970667358221220eeba4e29abce0ca57ba16aaf3ed2ed6bc73abc754e7e29c81023410857d67e6364736f6c63430008120033")
	simpleStorageAbi := `[
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "x",
					"type": "uint256"
				},
				{
					"internalType": "uint256",
					"name": "y",
					"type": "uint256"
				}
			],
			"name": "set",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "get",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				},
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "key",
					"type": "uint256"
				}
			],
			"name": "getStorageLocationForKey",
			"outputs": [
				{
					"internalType": "bytes32",
					"name": "",
					"type": "bytes32"
				}
			],
			"stateMutability": "pure",
			"type": "function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"name": "test",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`
	exres := self.Execute(self.Sender, big.NewInt(0), simpleStorageCode)
	ej, _ := json.Marshal(exres)
	fmt.Println(string(ej))
	self.ContractAddr = &exres.NewContractAddr

	self.abi, _ = abi.JSON(strings.NewReader(simpleStorageAbi))
}

func (self *ContractTest) Execute(from common.Address, value *big.Int, input []byte) vm.ExecutionResult {
	senderNonce := self.GetNonce(from)
	senderNonce.Add(senderNonce, big.NewInt(1))

	self.blk_n++
	self.st.BeginBlock(&vm.BlockInfo{})

	res := self.st.ExecuteTransaction(&vm.Transaction{
		Value:    value,
		To:       self.ContractAddr,
		From:     from,
		Input:    input,
		Gas:      10000000,
		GasPrice: big.NewInt(1),
		Nonce:    senderNonce,
	})

	self.st.EndBlock()
	self.st.Commit()
	return res
}

func (self *ContractTest) AdvanceBlock(author *common.Address, rewardsStats *rewards_stats.RewardsStats) (root common.Hash, ret *uint256.Int) {
	self.blk_n++
	if author == nil {
		self.st.BeginBlock(&vm.BlockInfo{})
	} else {
		self.st.BeginBlock(&vm.BlockInfo{Author: *author, GasLimit: 0, Time: 0, Difficulty: nil})
	}
	ret = self.st.DistributeRewards(rewardsStats)
	self.st.EndBlock()
	root = self.st.Commit()
	return
}

func (self *ContractTest) GetBalance(account common.Address) *big.Int {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	return bal_actual
}

func (self *ContractTest) GetNonce(account common.Address) *big.Int {
	nonce := big.NewInt(0)
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		nonce = account.Nonce
	})
	return nonce
}

func (self *ContractTest) GetDPOSReader() dpos.Reader {
	return self.SUT.DPOSReader(self.blk_n)
}

func (self *ContractTest) end() {
	self.statedb.Close()
	self.tc.Close()
}

func (self *ContractTest) pack(name string, args ...interface{}) []byte {
	packed, err := self.abi.Pack(name, args...)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return packed
}

func (self *ContractTest) unpack(v interface{}, name string, output []byte) error {
	err := self.abi.Unpack(v, name, output)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return err
}

func TestTrieVal(t *testing.T) {
	_, test := init_contract_test(t, CopyDefaultChainConfig())
	defer test.end()

	root, _ := test.AdvanceBlock(nil, nil)
	fmt.Println("state root", root.Hex())
	fmt.Println("test contract addr", test.ContractAddr.Hex())

	// for i := 100; i < 230; i++ {
	// 	code := test.pack("set", big.NewInt(int64(i)), big.NewInt(int64(i+100)))
	// 	// fmt.Println(common.Bytes2Hex(code))
	// 	test.Execute(test.Sender, BigZero, code)
	// }
	// test.AdvanceBlock(nil, nil, nil)

	state := state_db.ExtendedReader{Reader: test.statedb.GetBlockState(test.blk_n)}

	// state.GetAccountStorage(test.ContractAddr, &h, func(bytes []byte) {
	// 	fmt.Println("storage", common.Bytes2Hex(bytes))
	// })

	// state.ForEachStorage(test.ContractAddr, func(h *common.Hash, bytes []byte) {
	// 	// fmt.Println("storage", h.String(), common.Bytes2Hex(bytes))
	// 	fmt.Println("storage", h.String(), common.Bytes2Hex(bytes))
	// })

	fmt.Println()
	fmt.Println()
	storage_proof, _ := state.GetStorageProof(&root, test.ContractAddr)

	for i, r := range storage_proof {
		fmt.Println("proof", i, common.Bytes2Hex(r))
	}
	fmt.Println("stateRoot", root.Hex(), "from proof", common.Bytes2Hex(crypto.Keccak256(storage_proof[0])))
	//

	fmt.Println()
	fmt.Println()
	h := common.HexToHash("0x7673bcbb3401a7cbae68f81d40eea2cf35afdaf7ecd016ebf3f02857fcc1260a")
	res, _ := state.GetProof(test.ContractAddr, &h)
	for i, r := range res {
		fmt.Println("proof", i, common.Bytes2Hex(r))
	}

	// var acc state_db.Account
	// state.GetAccount(test.ContractAddr, func(a state_db.Account) {
	// 	acc = a
	// })
	// fmt.Println("account root", acc.StorageRootHash.Hex(), "from proof", common.Bytes2Hex(crypto.Keccak256(res[0])))
	// account := state_db.DecodeAccountFromTrie(common.Hex2Bytes("f8470180a0e6af15d077c6a71cd3ee33c674b003f5eccdbc8e9fda867020a37cf3a62bc970a03bb6c927aa8d73f4be3c479e49a5357adc0ff448b5bfea109aaa067d29d6403b82031c"))
	// aj, _ := json.Marshal(account)
	// fmt.Println(string(aj))

	// nextHash af60076e9315d9b41a8c42adeb78fef8551f24acb234f37d9d7537ef9ad2e78f
	// rlp node f8f18080a0be6aba9ac1c6352f17e80656c7f0b3953ec8ce0e1f2cfb05ef769bfc37d1d77580a0b7b49995dce6c474becec567a6c96c93da7e0990598ac4dc1affbcd3ad452453a010c7db9de7b9d84c39923dbd3949bce63056bfc5e6f0e8b9c4cab3463b6b878180a09e9e038f1f2f81d0bb0e1f416f50de622e9fcecadb4197c5c2bdce5088c7aa2380a05e1ea58069877ad278bf8365b9e3f5448a10f437314d1d2068d84d927b58c747a0ef4ffe20ccabaf1f61203f68e1d6e00cd25a4f54a782609104630224c2eafc7680a0fb21b3c21a8b1068229c1e1380f327c83bffd17a6036c867839c6eb6552248b880808080
	// hash 731f155159fc21ab77a156b7735406bd2510206021b4f3137637cd0c32a80721
}

type StorageProof struct {
	Key   common.Hash
	Value []byte
	Proof [][]byte
}

func (u StorageProof) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key   string   `json:"key"`
		Value string   `json:"value"`
		Proof []string `json:"proof"`
	}{
		Key:   u.Key.Hex(),
		Value: common.Bytes2Hex(u.Value),
		Proof: func() (ret []string) {
			for _, p := range u.Proof {
				ret = append(ret, common.Bytes2Hex(p))
			}
			return
		}(),
	})
}

type FullProof struct {
	Balance      big.Int
	CodeHash     common.Hash
	Nonce        big.Int
	StorageHash  common.Hash
	AccountProof [][]byte
	StorageProof []StorageProof
}

func (u FullProof) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Balance      string         `json:"balance"`
		CodeHash     string         `json:"codeHash"`
		Nonce        string         `json:"nonce"`
		StorageHash  string         `json:"storageHash"`
		AccountProof []string       `json:"accountProof"`
		StorageProof []StorageProof `json:"storageProof"`
	}{
		Balance:     u.Balance.String(),
		CodeHash:    u.CodeHash.Hex(),
		Nonce:       u.Nonce.String(),
		StorageHash: u.StorageHash.Hex(),
		AccountProof: func() (ret []string) {
			for _, p := range u.AccountProof {
				ret = append(ret, common.Bytes2Hex(p))
			}
			return
		}(),
		StorageProof: u.StorageProof,
	})
}

func GetProof(blk_state state_db.ExtendedReader, address common.Address, state_root common.Hash, keys []common.Hash) (ret FullProof) {
	blk_state.GetAccount(&address, func(acc state_db.Account) {
		ret.Balance = *acc.Balance
		ret.CodeHash = *acc.CodeHash
		ret.Nonce = *acc.Nonce
		ret.StorageHash = *acc.StorageRootHash
	})
	var err error
	ret.AccountProof, err = blk_state.GetStorageProof(&state_root, &address)
	if err != nil {
		fmt.Println("GetStorageProof error:", err)
	}
	for _, key := range keys {
		var proof StorageProof
		proof.Key = key
		proof.Proof, err = blk_state.GetProof(&address, &key)
		if err != nil {
			fmt.Println("GetProof error:", err)
		}
		blk_state.GetAccountStorage(&address, &key, func(bytes []byte) {
			proof.Value = bytes
		})

		ret.StorageProof = append(ret.StorageProof, proof)
	}
	return
}

func TestGetProof(t *testing.T) {
	_, test := init_contract_test(t, CopyDefaultChainConfig())
	defer test.end()

	root, _ := test.AdvanceBlock(nil, nil)
	fmt.Println("state root", root.Hex())
	fmt.Println("test contract addr", test.ContractAddr.Hex())

	// for i := 100; i < 230; i++ {
	// 	code := test.pack("set", big.NewInt(int64(i)), big.NewInt(int64(i+100)))
	// 	// fmt.Println(common.Bytes2Hex(code))
	// 	test.Execute(test.Sender, BigZero, code)
	// }
	// test.AdvanceBlock(nil, nil, nil)

	state := state_db.ExtendedReader{Reader: test.statedb.GetBlockState(test.blk_n)}
	// state.GetProof()
	keys := []common.Hash{
		common.HexToHash("0x7673bcbb3401a7cbae68f81d40eea2cf35afdaf7ecd016ebf3f02857fcc1260a"), // 100
		common.HexToHash("0x0bb0d0c2a399402027fb0eaada47a2c630983f3dd97f193c64f3e30465d04ec3"), // 101
		common.HexToHash("0x7ec3c2f200843fc90126ba954586fc6de68c1a3d419d6f0667702fd695602f05"), // 150
		common.HexToHash("0x46a4a9204e2252337cfce182401bbabede11720ab2d2e2330f66de6cfcb0b379"), // 170
		common.HexToHash("0xd83db53d400092e1cd810411bbe8320db49f103fdcd91dec3d07ef7ac3dacd1e"), // 181
		common.HexToHash("0xef407a61ad059ad1a9edcba0919f1209387c016d49079cb4d982b420eb78a186"), // 221
	}
	p := GetProof(state, *test.ContractAddr, root, keys)
	fmt.Println()
	fmt.Println()
	fmt.Println()
	pj, _ := json.Marshal(p)
	fmt.Println(string(pj))
	fmt.Println("state root", root.Hex())

}
