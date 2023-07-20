// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface SlashingInterface {
    event NewProof(address indexed author, address indexed validator, uint8 proof_type);
    event Jailed(address indexed validator, uint256 block);
    event Slashed(address indexed validator, uint256 amount);

    // Commit double voting malicious behaviour proof
    function commitDoubleVotingProof(
        address author,    // proof author
        address validator, // malicious validator
        bytes memory vote1,
        bytes memory vote2
    ) external;

    /**
     * @notice Returns true if validator is currently jailed due to malicious behaviour, otherwise false
     *
     * @param validator     validator account address
     **/
    function isJailed(address validator) external view returns (bool);
}
