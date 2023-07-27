// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface SlashingInterface {
    event NewProof(address indexed author, address indexed validator, uint8 proof_type);
    event Jailed(address indexed validator, uint256 block);
    event Slashed(address indexed validator, uint256 amount);

    // Commit double voting malicious behaviour proof
    function commitDoubleVotingProof(
        bytes memory vote1,
        bytes memory vote2
    ) external;

    /**
     * @notice Returns true if validator is currently jailed due to malicious behaviour, otherwise false
     *
     * @param validator validator's address
     **/
    function isJailed(address validator) external view returns (bool);

    struct JailInfo {
        uint256 jail_block;  // block until which is validator jailed
        bool is_jailed;      // flag if validator is currently jailed
        uint32 proofs_count; // number of malicious behaviour proofs
    }

    /**
     * @notice Returns validator's jail info - jail_block == 0 in case 
     *
     * @param validator validator's address
     **/
    function getJailInfo(address validator) external view returns (JailInfo memory info);

    struct MaliciousValidator {
        address validator;
        JailInfo jail_info;
    }

    /**
     * @notice Returns list of malicious validators
     *
     * @return validators Batch of N malicious validators
     **/
    function getMaliciousValidators()
        external
        view
        returns (MaliciousValidator[] memory);

    struct DoubleVotingProof {
        address proof_author; // author of the proof
        uint256 block;
        string vote1_hash; 
        string vote2_hash; 
        string tx_hash; 
    }

    /**
     * @notice Returns list of malicious validators
     * @param validator validator's address
     *
     * @return proofs double voting proofs
     **/
    function getDoubleVotingProofs(address validator)
        external
        view
        returns (DoubleVotingProof[] memory);
}
