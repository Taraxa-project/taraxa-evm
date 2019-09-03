package ethereum_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	evm "github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"
	"sync"
)

type EthereumVM struct {
	*base_vm.BaseVM
	EthereumVMConfig
	stateDB *state.StateDB
	mutex   sync.Mutex
}

func (this *EthereumVM) TransitionState(req *vm.StateTransitionRequest) (ret *vm.StateTransitionResult, err error) {
	defer concurrent.LockUnlock(&this.mutex)()
	ret = new(vm.StateTransitionResult)
	if req.Block.Number.Sign() == 0 {
		this.ApplyGenesis()
		ret.StateRoot = this.GenesisBlock.Root()
		return
	}
	if this.stateDB == nil || this.calculateRoot(req.Block) != req.BaseStateRoot {
		if this.stateDB, err = state.New(req.BaseStateRoot, this.ReadDB); err != nil {
			return
		}
	}
	this.applyHardForks(req.Block)
	gasPool := new(core.GasPool).AddGas(uint64(req.Block.GasLimit))
	for i, tx := range req.Block.Transactions {
		this.stateDB.Prepare(tx.Hash, req.Block.Hash, i)
		txResult := this.BaseVM.ExecuteTransaction(&base_vm.TransactionRequest{
			Transaction:      tx,
			BlockHeader:      &req.Block.BlockHeader,
			DB:               this.stateDB,
			OnEvmInstruction: evm.NoopExecutionController,
			GasPool:          gasPool,
			CheckNonce:       true,
			CanTransfer:      core.CanTransfer,
		})
		if err = txResult.ConsensusErr; err != nil {
			return
		}
		var intermediateRoot []byte
		if this.Genesis.Config.IsByzantium(req.Block.Number) {
			this.stateDB.Finalise(true)
		} else {
			intermediateRoot = this.calculateRoot(req.Block).Bytes()
		}
		ret.UsedGas += hexutil.Uint64(txResult.GasUsed)
		ethReceipt := types.NewReceipt(intermediateRoot, txResult.ContractErr != nil, uint64(ret.UsedGas))
		if tx.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, uint64(tx.Nonce))
		}
		ethReceipt.TxHash = tx.Hash;
		ethReceipt.GasUsed = txResult.GasUsed
		ethReceipt.Logs = this.stateDB.GetLogs(tx.Hash)
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, &vm.TaraxaReceipt{
			ReturnValue:     txResult.EVMReturnValue,
			ContractError:   txResult.ContractErr,
			EthereumReceipt: ethReceipt,
		})
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}
	this.applyMinerReward(req.Block)
	ret.StateRoot, err = this.stateDB.Commit(this.Genesis.Config.IsEIP158(req.Block.Number))
	return
}

func (this *EthereumVM) applyMinerReward(block *vm.Block) {
	if this.DisableMinerReward {
		return
	}
	var unclesMapped []*types.Header
	for _, uncle := range block.UncleBlocks {
		unclesMapped = append(unclesMapped, &types.Header{Number: uncle.Number.ToInt(), Coinbase: uncle.Miner})
	}
	ethash.AccumulateRewards(
		this.Genesis.Config,
		this.stateDB,
		&types.Header{Number: block.Number, Coinbase: block.Miner},
		unclesMapped)
}

func (this *EthereumVM) calculateRoot(block *vm.Block) common.Hash {
	return this.stateDB.IntermediateRoot(this.Genesis.Config.IsEIP158(block.Number))
}

func (this *EthereumVM) applyHardForks(block *vm.Block) (stateChanged bool) {
	chainConfig := this.Genesis.Config
	DAOForkBlock := chainConfig.DAOForkBlock
	if chainConfig.DAOForkSupport && DAOForkBlock != nil && DAOForkBlock.Cmp(block.Number) == 0 {
		misc.ApplyDAOHardFork(this.stateDB)
		return true
	}
	return false
}
