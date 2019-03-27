pragma solidity >=0.4.0;

contract SingleVariable {

    int value = 5;

    function set(int _value) public {
        value = _value;
    }

    function get() public view returns (int) {
        return value;
    }

}