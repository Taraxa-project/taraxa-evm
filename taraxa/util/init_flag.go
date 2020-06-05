package util

import "github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

type InitFlag struct {
	initialized bool
}

func (self *InitFlag) InitOnce() {
	self.initialized = assert.Holds(self.IsZero())
}

func (self *InitFlag) IsZero() bool {
	return !self.initialized
}
