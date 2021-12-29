package storage_accessor

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/stretchr/testify/assert"
)

func TestMapAt(t *testing.T) {
	// map stored at position 1 in storage
	// map[4]["0x8671A6B8d5781Db8920166c77B1D8749704062cF"]
	assert := assert.New(t)
	accessor := new(StorageAccessor)

	b := common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000001")
	// web3.sha3('00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000001', { encoding: 'hex' })
	// 0xedc95719e9a3b28dd8e80877cb5880a9be7de1a13fc8b05e7999683b6b567643
	h := keccak256.Hash(b)
	assert.Equal(*h, common.HexToHash("0xedc95719e9a3b28dd8e80877cb5880a9be7de1a13fc8b05e7999683b6b567643"))
	accessor = accessor.Field(1).MapAt(4)
	// map[4]
	assert.Equal(accessor.Key(), *h)

	address := common.HexToAddress("0x8671A6B8d5781Db8920166c77B1D8749704062cF").Hash()
	accessor = accessor.MapAtHash(address)
	// map[4]["0x8671A6B8d5781Db8920166c77B1D8749704062cF"]
	assert.Equal(*keccak256.Hash(address.Bytes(), h.Bytes()), accessor.Key())
}

func TestArrayAt(t *testing.T) {
	// array stored at position 2 in storage
	// array[2][2]
	assert := assert.New(t)
	accessor := new(StorageAccessor)

	accessor = accessor.Field(2).Array().At(0)
	h := *keccak256.Hash(common.BytesToHash(big.NewInt(2).Bytes()).Bytes())
	assert.Equal(accessor.Key(), h)

	accessor = accessor.At(2)
	h = h.Add(big.NewInt(2))
	assert.Equal(accessor.Key(), h)

	acc := *accessor
	acc = *acc.Array().At(2)
	accessor = accessor.Array().At(0).At(2)
	assert.Equal(acc.Key(), accessor.Key())

	h = *keccak256.Hash(h.Bytes())
	h = h.Add(big.NewInt(2))
	assert.Equal(h, acc.Key())
}

func TestPrepareValuesForCppTest(t *testing.T) {
	assert := assert.New(t)

	// contract Staking {
	//   struct DelegationData {
	//     // Total number of delegated coins
	//     uint256 delegatedCoins;
	//     // ??? What is this delegateePercents ?
	//     uint256 delegateePercents;
	//     // List of delegators
	//     IterableMap delegators;
	//   }
	//   // Iterable map that is used only together with the _delegators mapping as data holder
	//   struct IterableMap {
	//     // map of indexes to the list array
	//     // indexes are shifted +1 compared to the real indexes of this list, because 0 means non-existing element
	//     mapping(address => uint256) listIndex;
	//     // list of addresses
	//     address[]                   list;
	//   }
	//   uint256 a = 1000;
	//   mapping (address => DelegationData) private _delegators;
	//
	//   constructor() {
	//    _delegators[address(0x1337)].delegatedCoins = 0x12345;
	//    _delegators[address(0x1337)].delegateePercents = 0x3000;
	//    _delegators[address(0x1337)].delegators.listIndex[address(0x1234567890)] = 0x100011;
	//    _delegators[address(0x1337)].delegators.list.push(address(0x1234));
	//    _delegators[address(0x1337)].delegators.list.push(address(0xabcd));
	//    _delegators[address(0x1337)].delegators.list.push(address(0x12abcdef));
	//   }
	// }

	accessor := new(StorageAccessor)
	test_accessor := *accessor

	delegatorAddress := common.HexToHash("0x1337")
	// _delegators[address(0x1337)].delegatedCoins
	accessor = accessor.Field(1).MapAtHash(delegatorAddress).Struct().At(0)
	fmt.Println("const auto delegatedCoins = dev::jsToU256(\"" + accessor.Key().Hex() + "\"); // = 0x12345")

	{
		// _delegators[address(0x1337)].delegateePercents
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(1).Key()
		key := accessor.At(1).Key()
		assert.Equal(key, tk)
		fmt.Println("const auto delegateePercents = dev::jsToU256(\"" + key.Hex() + "\"); // = 0x3000")
	}
	{
		// _delegators[address(0x1337)].delegators.listIndex
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(2).Key()
		key := accessor.At(2).Key()
		assert.Equal(key, tk)
		fmt.Println("const auto delegators_listIndex = dev::jsToU256(\"" + key.Hex() + "\"); // = 0 map stores nothing")
		test_accessor.reset()
	}
	map_accessor := *accessor.At(2)
	{
		// _delegators[address(0x1337)].delegators.list
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(3).Key()
		key := accessor.At(3).Key()
		assert.Equal(key, tk)
		fmt.Println("const auto delegators_list = dev::jsToU256(\"" + key.Hex() + "\"); // size = 3")
		test_accessor.reset()
	}
	{
		accessor = accessor.At(3).Array()
		// _delegators[address(0x1337)].delegators.list[1]
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(3).Array().At(1).Key()
		key := accessor.At(1).Key()
		assert.Equal(accessor.At(1).Key(), tk)
		fmt.Println("const auto array_1 = dev::jsToU256(\"" + key.Hex() + "\"); // = 0xabcd")
		test_accessor.reset()
	}
	{
		// _delegators[address(0x1337)].delegators.list[0]
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(3).Array().At(0).Key()
		key := accessor.At(0).Key()
		assert.Equal(key, tk)
		fmt.Println("const auto array_0 = dev::jsToU256(\"" + key.Hex() + "\"); // = 0x1234")
		test_accessor.reset()
	}
	{
		// _delegators[address(0x1337)].delegators.list[2]
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).At(3).Array().At(2).Key()
		key := accessor.At(2).Key()
		assert.Equal(accessor.At(2).Key(), tk)
		fmt.Println("const auto array_2 = dev::jsToU256(\"" + key.Hex() + "\"); // = 0x12abcdef")
		test_accessor.reset()
	}
	{
		// _delegators[address(0x1337)].delegators.listIndex["0x1234567890"]
		tk := test_accessor.Field(1).MapAtHash(delegatorAddress).Struct().Field(2).MapAtHash(common.HexToHash("0x1234567890")).Key()
		map_key := map_accessor.MapAtHash(common.HexToHash("0x1234567890")).Key()
		assert.Equal(map_key, tk)
		fmt.Println("const auto map_at_1234567890 = dev::jsToU256(\"" + map_key.Hex() + "\"); // = 0x100011")
	}
}
