// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface DposInterface {
    event Delegated(address indexed delegator, address indexed validator, uint256 amount);
    event Undelegated(address indexed delegator, address indexed validator, uint256 amount);
    event UndelegateConfirmed(address indexed delegator, address indexed validator, uint256 amount);
    event UndelegateCanceled(address indexed delegator, address indexed validator, uint256 amount);
    event UndelegatedV2(address indexed delegator, address indexed validator, uint64 indexed undelegation_id, uint256 amount);
    event UndelegateConfirmedV2(address indexed delegator, address indexed validator, uint64 indexed undelegation_id, uint256 amount);
    event UndelegateCanceledV2(address indexed delegator, address indexed validator, uint64 indexed undelegation_id, uint256 amount);
    event Redelegated(address indexed delegator, address indexed from, address indexed to, uint256 amount);
    event RewardsClaimed(address indexed account, address indexed validator, uint256 amount);
    event CommissionRewardsClaimed(address indexed account, address indexed validator, uint256 amount);
    event CommissionSet(address indexed validator, uint16 commission);
    event ValidatorRegistered(address indexed validator);
    event ValidatorInfoSet(address indexed validator);

    struct ValidatorBasicInfo {
        // Total number of delegated tokens to the validator
        uint256 total_stake;
        // Validator's reward from delegators rewards commission
        uint256 commission_reward;
        // Validator's commission - max value 10000(precision up to 0.01%)
        uint16 commission;
        // Block number of last commission change
        uint64 last_commission_change;
        // Number of ongoing undelegations from the validator
        uint16 undelegations_count;
        // Validator's owner account
        address owner;
        // Validators description/name
        string description;
        // Validators website endpoint
        string endpoint;
    }

    // Retun value for getValidators method
    struct ValidatorData {
        address account;
        ValidatorBasicInfo info;
    }

    // Delegator data
    struct DelegatorInfo {
        // Number of tokens that were staked
        uint256 stake;
        // Number of tokens that were rewarded
        uint256 rewards;
    }

    // Retun value for getDelegations method
    struct DelegationData {
        // Validator's(in case of getDelegations) or Delegator's (in case of getValidatorDelegations) account address
        address account;
        // Delegation info
        DelegatorInfo delegation;
    }

    // Retun value for getUndelegations method
    struct UndelegationData {
        // Number of tokens that were locked
        uint256 stake;
        // block number when it will be unlocked
        uint64 block;
        // Validator address
        address validator;
        // Flag if validator still exists - in case he has 0 stake and 0 rewards, validator is deleted from memory & db
        bool validator_exists;
    }

    // Retun value for getUndelegationsV2 method
    struct UndelegationV2Data {
        // Undelegation data
        UndelegationData undelegation_data;
        // Undelegation id
        uint64 undelegation_id;
    }

    // Delegates tokens to specified validator
    function delegate(address validator) external payable;

    // Undelegates <amount> of tokens from specified validator - creates undelegate request
    // Note: deprecated (pre cornus hardfork) - use undelegateV2 instead
    function undelegate(address validator, uint256 amount) external;

    // Undelegates <amount> of tokens from specified validator - creates undelegate request and returns unique undelegation_id <per delegator>
    function undelegateV2(address validator, uint256 amount) external returns (uint64 undelegation_id);

    // Confirms undelegate request
    // Note: deprecated (pre cornus hardfork) - use confirmUndelegateV2 instead
    function confirmUndelegate(address validator) external;

    // Confirms undelegate request with <undelegation_id> from <validator>
    function confirmUndelegateV2(address validator, uint64 undelegation_id) external;

    // Cancel undelegate request
    // Note: deprecated (pre cornus hardfork) - use confirmUndelegateV2 instead
    function cancelUndelegate(address validator) external;

    // Cancel undelegate request with <undelegation_id> from <validator>
    function cancelUndelegateV2(address validator, uint64 undelegation_id) external;

    // Redelegates <amount> of tokens from one validator to the other
    function reDelegate(address validator_from, address validator_to, uint256 amount) external;

    // Claims staking rewards from <validator>
    function claimRewards(address validator) external;

    /**
     * @notice Claims staking rewards from all validators (limited by max dag block gas limit) that caller has delegated to
     *
     */
    function claimAllRewards() external;

    // Claims tokens from validator's commission rewards
    function claimCommissionRewards(address validator) external;

    // Registers new validator - validator also must delegate to himself, he can later withdraw his delegation
    function registerValidator(
        address validator,
        bytes memory proof,
        bytes memory vrf_key,
        uint16 commission,
        string calldata description,
        string calldata endpoint
    ) external payable;

    /**
     * @notice Sets some of the static validator details.
     *
     * @param description   New description (e.g name, short purpose description, etc...)
     * @param endpoint      New endpoint, might be a validator's website
     *
     */
    function setValidatorInfo(address validator, string calldata description, string calldata endpoint) external;

    // Sets validator's commission [%] * 100 so 1% is 100 & 10% is 1000
    function setCommission(address validator, uint16 commission) external;

    // TODO: these 4 methods below can be all replaced by "getValidator" and "getValidators" calls, but it should be
    //       considered in terms of performance, etc...

    // Returns true if acc is eligible validator, otherwise false
    function isValidatorEligible(address validator) external view returns (bool);

    // Returns all validators eligible votes counts
    function getTotalEligibleVotesCount() external view returns (uint64);

    // Returns specified validator eligible votes count
    function getValidatorEligibleVotesCount(address validator) external view returns (uint64);

    // Returns validator basic info (everything except list of his delegators)
    function getValidator(address validator) external view returns (ValidatorBasicInfo memory validator_info);

    function getValidators(uint32 batch) external view returns (ValidatorData[] memory validators, bool end);

    /**
     * @notice Returns list of validators owned by an address
     *
     * @param owner        Owner address
     * @param batch        Batch number to be fetched. If the list is too big it cannot return all validators in one call. Instead, users are fetching batches of 100 account at a time
     *
     * @return validators  Batch of N validators basic info
     * @return end         Flag if there are no more accounts left. To get all accounts, caller should fetch all batches until he sees end == true
     *
     */
    function getValidatorsFor(address owner, uint32 batch)
        external
        view
        returns (ValidatorData[] memory validators, bool end);

    /**
     * @notice Returns total delegation for specified delegator
     *
     * @param delegator Delegator account address
     *
     * @return total_delegation amount that was delegated
     *
     */
    function getTotalDelegation(address delegator) external view returns (uint256 total_delegation);

    /**
     * @notice Returns list of delegations for specified delegator - which validators delegator delegated to
     *
     * @param delegator     delegator account address
     * @param batch         Batch number to be fetched. If the list is too big it cannot return all delegations in one call. Instead, users are fetching batches of 50 delegations at a time
     *
     * @return delegations  Batch of N delegations
     * @return end          Flag if there are no more delegations left. To get all delegations, caller should fetch all batches until he sees end == true
     *
     */
    function getDelegations(address delegator, uint32 batch)
        external
        view
        returns (DelegationData[] memory delegations, bool end);

    /**
     * @notice Returns list of undelegations for specified delegator
     *
     * @param delegator       delegator account address
     * @param batch           Batch number to be fetched. If the list is too big it cannot return all undelegations in one call. Instead, users are fetching batches of 50 undelegations at a time
     *
     * @return undelegations  Batch of N undelegations
     * @return end            Flag if there are no more undelegations left. To get all undelegations, caller should fetch all batches until he sees end == true
     *
     */
    function getUndelegations(address delegator, uint32 batch)
        external
        view
        returns (UndelegationData[] memory undelegations, bool end);

   /**
     * @notice Returns list of V2 undelegations for specified delegator
     *
     * @param delegator       delegator account address
     * @param batch           Batch number to be fetched. If the list is too big it cannot return all undelegations in one call. Instead, users are fetching batches of 50 undelegations at a time
     *
     * @return undelegations_v2  Batch of N undelegations
     * @return end            Flag if there are no more undelegations left. To get all undelegations, caller should fetch all batches until he sees end == true
     *
     */
    function getUndelegationsV2(address delegator, uint32 batch)
        external
        view
        returns (UndelegationV2Data[] memory undelegations_v2, bool end);

     /**
     * @notice Returns V2 undelegation for specified delegator, validator & and undelegation_id
     *
     * @param delegator        delegator account address
     * @param validator        validator account address
     * @param undelegation_id  undelegation id
     *
     * @return undelegation_v2
     */
    function getUndelegationV2(address delegator, address validator, uint64 undelegation_id)
        external
        view
        returns (UndelegationV2Data memory undelegation_v2);
}
