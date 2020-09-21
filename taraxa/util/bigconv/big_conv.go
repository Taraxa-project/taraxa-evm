package bigconv

import (
	"math/big"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type BigConv struct {
	buf  common.Hash
	addr *common.Address
}

func (self *BigConv) Init() *BigConv {
	self.addr = (*common.Address)(bin.UnsafeAdd(unsafe.Pointer(&self.buf), uintptr(len(self.buf)-common.AddressLength)))
	return self
}

func (self *BigConv) ToHash(b *big.Int) (ret *common.Hash) {
	switch l := b.BitLen(); {
	case l <= common.HashLength*8:
		ret = &self.buf
		b.FillBytes(self.buf[:])
	default:
		ret = new(common.Hash).SetBytes(b.Bytes())
	}
	return
}

func (self *BigConv) ToAddr(b *big.Int) (ret *common.Address) {
	switch l := b.BitLen(); {
	case l <= common.AddressLength*8:
		ret = self.addr
		b.FillBytes(self.addr[:])
	case l <= common.HashLength*8:
		ret = self.addr
		b.FillBytes(self.buf[:])
	default:
		ret = new(common.Address)
		ret.SetBytes(b.Bytes())
	}
	return
}
