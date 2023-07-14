package dpos

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

// IterableMap storage fields keys - relative to the prefix from Init function
var (
	field_accounts       = []byte{0}
	field_accounts_count = []byte{1}
	field_accounts_pos   = []byte{2}
)

// IterableMap storage wrapper
type IterableMap struct {
	storage                     *contract_storage.StorageWrapper
	accounts_storage_prefix     []byte       // accounts are stored under "accounts_storage_prefix + pos" key
	accounts_count_storage_key  *common.Hash // accounts count is stored under accounts_count_storage_key
	accounts_pos_storage_prefix []byte       // accounts positions are stored under "accounts_pos_storage_prefix + address" key
}

// Inits iterbale map with prefix, so multiple iterbale maps can coexists thanks to different prefixes
func (self *IterableMap) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor
	self.accounts_storage_prefix = append(prefix, field_accounts...)
	self.accounts_count_storage_key = contract_storage.Stor_k_1(prefix, field_accounts_count)
	self.accounts_pos_storage_prefix = append(prefix, field_accounts_pos...)
}

// Checks is account exists in iterable map
func (self *IterableMap) AccountExists(account *common.Address) bool {
	acc_exists, _ := self.accountExists(account)
	return acc_exists
}

// Creates account from iterable map
func (self *IterableMap) CreateAccount(account *common.Address) bool {
	if acc_exists, _ := self.accountExists(account); acc_exists {
		panic("Account already exists")
	}

	// Gets keys array length
	accounts_count := self.GetCount()

	// Accounts positions are shifetd + 1, account 0 is saved on pos 1, etc... pos 0 is reserved for non-existent account
	new_account_pos := accounts_count + 1

	// Saves new account into the accounts array with key -> self.accounts_storage_prefix + pos
	accounts_k := contract_storage.Stor_k_1(self.accounts_storage_prefix, uint32ToBytes(new_account_pos))
	self.storage.Put(accounts_k, account.Bytes())

	// Save position of ney item in accounts array into the accounts pos mapping
	accounts_pos_k := contract_storage.Stor_k_1(self.accounts_pos_storage_prefix, account[:])
	self.storage.Put(accounts_pos_k, uint32ToBytes(new_account_pos))

	// Saves new accounts count
	self.storage.Put(self.accounts_count_storage_key, uint32ToBytes(accounts_count+1))

	return true
}

// Removes account from iterable map, returns number of left accounts in the iterbale map
func (self *IterableMap) RemoveAccount(account *common.Address) uint32 {
	// Gets accounts count
	accounts_count := self.GetCount()

	// There are no accounts saved in storage
	if accounts_count == 0 {
		panic("Unable to delete account " + account.String() + ". No accounts in iterable map")
	}

	// Checks if account to be deleted exists
	acc_exists, delete_acc_pos := self.accountExists(account)
	if acc_exists == false {
		panic("Account does not exist")
	}
	delete_acc_address_at_pos_k := contract_storage.Stor_k_1(self.accounts_storage_prefix, uint32ToBytes(delete_acc_pos))
	delete_acc_pos_k := contract_storage.Stor_k_1(self.accounts_pos_storage_prefix, account[:])

	// Account to be deleted is saved on the last position
	if delete_acc_pos == accounts_count {
		self.storage.Put(delete_acc_address_at_pos_k, nil)
		self.storage.Put(delete_acc_pos_k, nil)
		self.storage.Put(self.accounts_count_storage_key, uint32ToBytes(accounts_count-1))

		return accounts_count - 1
	}

	// There is more accounts saved and account to be deleted is somewhere in the middle

	// Positions are shifted +1 because pos == 0 is reserved to indicate non-existent element
	last_acc_address_at_pos_k := contract_storage.Stor_k_1(self.accounts_storage_prefix, uint32ToBytes(accounts_count))

	last_acc_address := common.ZeroAddress
	self.storage.Get(last_acc_address_at_pos_k, func(bytes []byte) {
		last_acc_address = common.BytesToAddress(bytes)
	})
	if last_acc_address == common.ZeroAddress {
		// This should never happen
		panic("Unable to delete account " + account.String() + ". Account not found")
	}

	last_acc_pos_k := contract_storage.Stor_k_1(self.accounts_pos_storage_prefix, last_acc_address[:])

	// Swap account to be deleted with the last item
	self.storage.Put(last_acc_pos_k, uint32ToBytes(delete_acc_pos))
	self.storage.Put(delete_acc_address_at_pos_k, last_acc_address.Bytes())

	self.storage.Put(delete_acc_pos_k, nil)
	self.storage.Put(last_acc_address_at_pos_k, nil)
	self.storage.Put(self.accounts_count_storage_key, uint32ToBytes(accounts_count-1))

	return accounts_count - 1
}

func (self *IterableMap) GetAccounts(batch uint32, count uint32) (result []common.Address, end bool) {
	// Gets accounts count
	accounts_count := self.GetCount()

	// No accounts in iterable map
	if accounts_count == 0 {
		end = true
		return
	}

	requested_idx_start := batch * count
	requested_idx_end := (batch + 1) * count

	// Invalid batch provided - there is not so many accounts in iterbale map
	if requested_idx_start >= accounts_count {
		end = true
		return
	}

	if accounts_count <= requested_idx_end {
		result = make([]common.Address, accounts_count-requested_idx_start)
		end = true
	} else {
		result = make([]common.Address, count)
		end = false
	}

	// Start with index == 1, there is nothing saved on index == 0 as it is reserved to indicate non-existent account
	var idx uint32
	for idx = uint32(requested_idx_start + 1); idx <= requested_idx_end && idx <= accounts_count; idx++ {
		accounts_k := contract_storage.Stor_k_1(self.accounts_storage_prefix, uint32ToBytes(idx))

		account := common.ZeroAddress
		self.storage.Get(accounts_k, func(bytes []byte) {
			account = common.BytesToAddress(bytes)
		})

		if account == common.ZeroAddress {
			// This should never happen
			panic("Unable to find account " + account.String())
		}

		result[(idx-1)%count] = account
	}

	return
}

// Returns number of stored items
func (self *IterableMap) GetCount() (count uint32) {
	count = 0
	self.storage.Get(self.accounts_count_storage_key, func(bytes []byte) {
		count = bytesToUint32(bytes)
	})

	return
}

// Checks is account exists in iterable map
// If account exists <true, position> it returned, otheriwse <false, 0>
func (self *IterableMap) accountExists(account *common.Address) (acc_exists bool, acc_pos uint32) {
	pos_k := contract_storage.Stor_k_1(self.accounts_pos_storage_prefix, account[:])
	acc_pos = 0

	self.storage.Get(pos_k, func(bytes []byte) {
		acc_pos = bytesToUint32(bytes)
	})

	// pos == 0 means non-existent account
	acc_exists = (acc_pos != 0)
	return
}

func uint32ToBytes(val uint32) []byte {
	r := make([]byte, 4)
	for i := uint32(0); i < 4; i++ {
		r[i] = byte((val >> (8 * i)) & 0xff)
	}
	return r
}

func bytesToUint32(val []byte) uint32 {
	r := uint32(0)
	for i := uint32(0); i < 4; i++ {
		r |= uint32(val[i]) << (8 * i)
	}
	return r
}
