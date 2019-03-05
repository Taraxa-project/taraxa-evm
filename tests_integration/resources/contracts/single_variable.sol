pragma solidity >=0.4.0;

contract single_variable {

    uint value = 2;

    function set(uint _value) public {
        value = _value;
    }

    function get() public view returns (uint) {
        return value;
    }

}