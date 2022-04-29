// (c) 2022-2023, Taraxa, Inc. All rights reserved.
// SPDX-License-Identifier: MIT

pragma solidity >=0.8.0;

interface DposInterface {
    // Delegates tokens to specified validator
    function delegate(address validator) external payable;

    // Undelegates <amount> of tokens from specified validator - creates undelegate request
    function undelegate(address validator, uint256 amount) external;

    // Confirms undelegate request
    function confirmUndelegate(address validator) external;

    // Redelegates <amount> of tokens from one validator to the other
    function reDelegate(address validator_from, address validator_to, uint256 amount) external;

    // Claims tokens from staking rewards based on delegation to <validator>
    function claimRewards(address validator) external;

    // Claims tokens from validator's commission rewards
    function claimCommissionRewards() external;

    // Registers new validator - validator also must delegate to himself, he can later withdraw his delegation
    function registerValidator(uint16 commission, string calldata description, string calldata endpoint) external payable;

    /**
     * @notice Sets some of the static validator details.
     *
     * @param description   New description (e.g name, short purpose description, etc...)
     * @param endpoint      New endpoint, might be a validator's website
     **/
    function setValidatorInfo(string calldata description, string calldata endpoint) external;

    // Sets validator's commission [%] * 100 so 1% is 100 & 10% is 1000
    function setCommission(uint16 commission) external;


    // TODO: these 4 methods below can be all replaced by "getValidator" and "getValidators" calls, but it should be
    //       considered in terms of performance, etc...

    // Returns true if acc is eligible validator, otherwise false
    // TODO: we need block_num when calling this method from node, but what if we call it from some external contract ?
    function isValidatorEligible(address validator) external view returns (bool);

    // Returns eligible validators counts
    // TODO: we need block_num when calling this method from node, but what if we call it from some external contract ?
    function getTotalEligibleValidatorsCount() external view returns (uint16);

    // Returns all validators eligible votes counts
    // TODO: we need block_num when calling this method from node, but what if we call it from some external contract ?
    function getTotalEligibleVotesCount() external view returns (uint32);

    // Returns specified validator eligible votes count
    // TODO: we need block_num when calling this method from node, but what if we call it from some external contract ?
    function getValidatorEligibleVotesCount(address validator) external view returns (uint32);


    struct ValidatorBasicInfo {
        // Total number of delegated tokens to the validator
        uint256 total_stake;

        // Validator's commission - max value 1000(precision up to 0.1%)
        uint16 commission;

        // Validator's reward from delegators rewards commission
        uint256 commission_reward;

        // Validators description
        // TODO: optional - might not be needed
        string description;

        // Validators website endpoint
        // TODO: optional - might not be needed
        string endpoint;
    }

    // Returns validator basic info (everything except list of his delegators)
    function getValidator(address validator) view external returns (ValidatorBasicInfo memory);

    // Retun value for getValidators method
    struct ValidatorData {
      address account;
      ValidatorBasicInfo info;
    }

    /**
     * @notice Returns list of registered validators
     *
     * @param batch        Batch number to be fetched. If the list is too big it cannot return all validators in one call. Instead, users are fetching batches of 100 account at a time
     *
     * @return validators  Batch of 100 validators basic info
     * @return count       How many accounts are returned in specified batch
     * @return end         Flag if there are no more accounts left. To get all accounts, caller should fetch all batches until he sees end == true
     **/
    function getValidators(uint16 batch) view external returns (ValidatorData[100] memory validators, uint16 count, bool end);


    struct UndelegateRequest {
        // Block num, during which UndelegateRequest can be confirmed - during creation it is
        // set to block.number + STAKE_UNLOCK_PERIOD
        uint256 eligible_block_num;

        // Amount of tokens to be unstaked
        uint256 amount;
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
        // Validator's(in case of getDelegatorDelegations) or Delegator's (in case of getValidatorDelegations) account address
        address account;

        // Delegation info
        DelegatorInfo delegation;
    }

    /**
     * @notice Returns list of delegations for specified delegator - which validators delegator delegated to
     *
     * @param delegator     delegator account address
     * @param batch         Batch number to be fetched. If the list is too big it cannot return all delegations in one call. Instead, users are fetching batches of 50 delegations at a time
     *
     * @return delegations  Batch of 50 delegations
     * @return count        How many delegations are returned in specified batch
     * @return end          Flag if there are no more delegations left. To get all delegations, caller should fetch all batches until he sees end == true
     **/
    function getDelegatorDelegations(address delegator, uint32 batch) view external returns (DelegationData[50] memory delegations, uint32 count, bool end);


    /**
     * @notice Returns list of delegations for specified validator - which delegators delegated to specified validator
     *
     * @param validator     validator account addres
     * @param batch         Batch number to be fetched. If the list is too big it cannot return all delegations in one call. Instead, users are fetching batches of 50 delegations at a time
     *
     * @return delegations  Batch of 50 delegations
     * @return count        How many delegations are returned in specified batch
     * @return end          Flag if there are no more delegations left. To get all delegations, caller should fetch all batches until he sees end == true
     **/
    function getValidatorDelegations(address validator, uint32 batch) view external returns (DelegationData[50] memory delegations, uint32 count, bool end);
}