package tests

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Taraxa-project/taraxa-evm/common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/files"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type TestCtx struct {
	*testing.T
	Assert   assert.Assertions
	data_dir string
}

func NewTestCtx(t *testing.T) (ret TestCtx) {
	ret.T = t
	ret.Assert = *assert.New(t)
	return
}

func (self *TestCtx) Close() {
	if len(self.data_dir) != 0 {
		files.RemoveAll(self.data_dir)
	}
}

func (self *TestCtx) DataDir() string {
	if len(self.data_dir) != 0 {
		return self.data_dir
	}
	_, test_file_path, _, _ := runtime.Caller(1)
	h := keccak256.HashAndReturnByValue(bin.BytesView(test_file_path), bin.BytesView(self.Name()))
	self.data_dir = files.CreateDirectoriesClean(os.TempDir(), h.Hex())
	return self.data_dir
}

func SimpleAddr(i int) (ret common.Address) {
	ret.SetBytes(bin.ENC_b_endian_compact_64_1(uint64(i)))
	return
}

func Noop(...interface{}) {}
