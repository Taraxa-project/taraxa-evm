// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface SlashingInterface {
    // Malicious behaviour types
    // uint8 DOUBLE_VOTING = 1
    event Jailed(
        address indexed validator,
        uint64 indexed start_block,
        uint64 indexed end_block,
        uint8 malicious_behaviour_type
    );

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

    /**
     * @notice Returns list of jailed validators
     *
     * @return list of jailed validators
     */
    function getJailedValidators() external view returns (address[] memory);
}
