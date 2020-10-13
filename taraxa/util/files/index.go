package files

import (
	"os"
	"path"
	"runtime"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

func ThisDirRelPath(path_segments ...string) string {
	_, ret, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(ret), path.Join(path_segments...))
}

func CreateDirectories(path_segments ...string) string {
	p := path.Join(path_segments...)
	util.PanicIfNotNil(os.MkdirAll(p, os.ModePerm))
	return p
}

func CreateDirectoriesClean(path_segments ...string) string {
	return CreateDirectories(RemoveAll(path_segments...))
}

func RemoveAll(path_segments ...string) string {
	p := path.Join(path_segments...)
	util.PanicIfNotNil(os.RemoveAll(p))
	return p
}
