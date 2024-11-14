package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	dpos_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/solidity"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition/op_stack"
)

func (st *StateTransition) applyHFChanges(blk_info *vm.BlockInfo) {
	blk_n := st.BlockNumber()

	if st.dpos_contract != nil {
		st.dpos_contract.Register(st.evm.RegisterPrecompiledContract)
		if st.chain_config.Hardforks.IsOnAspenHardforkPartOne(blk_n) {
			acc := st.evm_state.GetAccount(dpos.ContractAddress())
			if acc.GetCodeSize() == 0 {
				acc.SetCode(dpos_sol.AspenDposImplBytecode)
			}
		}
		if st.chain_config.Hardforks.IsCornusHardfork(blk_n) {
			acc := st.evm_state.GetAccount(dpos.ContractAddress())
			acc.SetCode(dpos_sol.CornusDposImplBytecode)
		}
	}

	if st.slashing_contract != nil && st.chain_config.Hardforks.IsOnMagnoliaHardfork(blk_n) {
		st.slashing_contract.Register(st.evm.RegisterPrecompiledContract)
	}

	if st.chain_config.Hardforks.IsCornusHardfork(blk_n) {
		for acc, byteCode := range op_stack.OpPrecompiles {
			acc := st.evm_state.GetAccount(&acc)
			acc.SetCode(byteCode)
		}
	}
}
