package dpos

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	dpos_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/solidity"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/holiman/uint256"
)

// GetOldClaimAllRewardsABI returns the *old* ABI method for claiming all rewards in the DPOS contract.
// It should be there, so we don't have a different result during the syncing. And it is hardcoded because we don't need it in the actual interface.
// If the block number is part of the Aspen hardfork, it returns nil.
// If the input matches the specified hex value, it returns the ABI method for claiming all rewards.
func (self *Contract) GetOldClaimAllRewardsABI(input []byte, blockNum types.BlockNum) *abi.Method {
	if bytes.Equal(input[0:4], common.FromHex("0x09b72e00")) {
		method := new(abi.Method)
		err := json.Unmarshal([]byte(`{
			"name": "claimAllRewards",
			"stateMutability": "nonpayable",
			"type": "function",
			"inputs": [
				{
					"internalType": "uint32",
					"name": "batch",
					"type": "uint32"
				}
			],
			"outputs": [
				{
					"internalType": "bool",
					"name": "end",
					"type": "bool"
				}
			]
		}`), method)

		if err != nil {
			return nil
		}
		return method
	}
	return nil
}

// Pays off accumulated rewards back to delegator address from multiple validators at a time
// NOTE this is old PRE-ASPEN HF version
func (self *Contract) claimAllRewardsPreAspenHF(ctx vm.CallFrame, block types.BlockNum, args dpos_sol.ClaimAllRewardsArgs) (end bool, err error) {
	delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(ctx.CallerAccount.Address(), args.Batch, ClaimAllRewardsMaxCount)
	var tmp_claim_rewards_args dpos_sol.ValidatorAddressArgs
	for _, validator_address := range delegator_validators_addresses {
		tmp_claim_rewards_args.Validator = validator_address

		tmp_err := self.claimRewards(ctx, block, tmp_claim_rewards_args)
		if tmp_err != nil {
			err = util.ErrorString(tmp_err.Error() + " -> validator: " + validator_address.String())
			return
		}
	}

	err = nil
	return
}

func (self *Contract) fixRedelegateBlockNumFunc(block_num uint64) {
	for _, redelegation := range self.cfg.Hardforks.Redelegations {
		delegation := self.delegations.GetDelegation(&redelegation.Delegator, &redelegation.Validator)

		val := self.validators.GetValidator(&redelegation.Validator)

		fmt.Println("Applying HF on validator", redelegation.Validator.String(), "delegator", redelegation.Delegator.String())

		state, _ := self.state_get(redelegation.Validator[:], BlockToBytes(delegation.LastUpdated))
		wrong_state, _ := self.state_get(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
		if wrong_state != nil || state == nil {
			panic("HF on wrong account")
		}

		fmt.Println("Fixing block from", val.LastUpdated, "to", delegation.LastUpdated)

		// Corrected block num
		val.LastUpdated = delegation.LastUpdated
		val.TotalStake = bigutil.Sub(val.TotalStake, redelegation.Amount)
		self.validators.ModifyValidator(self.isOnMagnoliaHardfork(block_num), &redelegation.Validator, val)
	}
}

func (self *Contract) processBlockReward(block_num uint64, blocks_per_year *uint256.Int) *uint256.Int {
	if self.cfg.Hardforks.IsOnAspenHardforkPartTwo(block_num) {
		if self.total_supply == nil {
			self.total_supply = self.yield_curve.CalculateTotalSupply(self.minted_tokens)
			self.saveTotalSupplyDb()

			// Erase minted_tokens from db as it is no longer needed
			self.eraseMintedTokensDb()
		}

		blockReward, yield := self.yield_curve.CalculateBlockReward(blocks_per_year, self.amount_delegated, self.total_supply)

		// Save current yield - it changes every block as total_supply is growing every block
		self.saveYieldDb(yield.Uint64())
		return blockReward
	}
	return nil
}

// func (self *Contract) bambooHFRedelegation(block_num uint64) {
// 	for _, redelegation := range self.cfg.Hardforks.BambooHf.Redelegations {
// 		val := self.validators.GetValidator(&redelegation.Validator)
// 		if val == nil {
// 			panic("Validator not found")
// 		}
// 		val_rewards := self.validators.GetValidatorRewards(&redelegation.Validator)
// 		if val_rewards == nil {
// 			panic("Validator rewards not found")
// 		}

// 		fmt.Println("Applying Bamboo HF on validator", redelegation.Validator.String(), "amount", redelegation.Amount.String())
// 		val.TotalStake = bigutil.Sub(val.TotalStake, redelegation.Amount)

// 		if val.TotalStake.Cmp(big.NewInt(0)) == 0 && val_rewards.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0 {
// 			self.validators.DeleteValidator(&redelegation.Validator)
// 			fmt.Println("Deleted validator", redelegation.Validator.String())
// 			state, stake_k := self.state_get(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
// 			if state != nil {
// 				self.state_put(&stake_k, nil)
// 			}
// 		} else {
// 			old_state := self.state_get_and_decrement(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
// 			state, state_k := self.state_get(redelegation.Validator[:], BlockToBytes(block_num))
// 			if state == nil {
// 				state = new(State)
// 				if val.TotalStake.Cmp(big.NewInt(0)) > 0 {
// 					state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(val_rewards.RewardsPool, val.TotalStake))
// 				} else {
// 					state.RewardsPer1Stake = old_state.RewardsPer1Stake
// 				}

// 				val_rewards.RewardsPool = big.NewInt(0)
// 				val.LastUpdated = block_num
// 				state.Count++
// 			}
// 			self.state_put(&state_k, state)
// 			self.validators.ModifyValidator(self.isMagnoliaHardfork(block_num), &redelegation.Validator, val)
// 			self.validators.ModifyValidatorRewards(&redelegation.Validator, val_rewards)
// 			fmt.Println("Updated validator", redelegation.Validator.String())
// 		}
// 	}
// }
