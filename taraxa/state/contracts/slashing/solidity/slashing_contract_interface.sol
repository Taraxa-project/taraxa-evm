// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface SlashingInterface {
    event Jailed(address indexed validator, uint64 block);

    // Commit double voting malicious behaviour proof
    function commitDoubleVotingProof(
        bytes memory vote_a,
        bytes memory vote_b
    ) external;

    /**
     * @notice Returns validator's jail info - jail_block == 0 in case 
     *
     * @param validator validator's address
     **/
    function getJailBlock(address validator) external view returns (uint64);
}
