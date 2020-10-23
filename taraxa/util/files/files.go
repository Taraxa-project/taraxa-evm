package files

import (
	"os"
	"path"
	"runtime"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	cpy "github.com/otiai10/copy"
)

func ThisDirRelPath(path_segments ...string) string {
	_, ret, _, _ := runtime.Caller(1)
	return Path(path.Dir(ret), Path(path_segments...))
}

func CreateDirectories(path_segments ...string) string {
	p := Path(path_segments...)
	util.PanicIfNotNil(os.MkdirAll(p, os.ModePerm))
	return p
}

func CreateDirectoriesClean(path_segments ...string) string {
	return CreateDirectories(RemoveAll(path_segments...))
}

func RemoveAll(path_segments ...string) string {
	p := Path(path_segments...)
	util.PanicIfNotNil(os.RemoveAll(p))
	return p
}

func Path(path_segments ...string) string {
	tmp := make([]string, len(path_segments))
	for i, s := range path_segments {
		if s == "~" {
			var err error
			s, err = os.UserHomeDir()
			util.PanicIfNotNil(err)
		}
		tmp[i] = s
	}
	return path.Join(tmp...)
}

func Exists(path_segments ...string) bool {
	_, err := os.Stat(Path(path_segments...))
	return !os.IsNotExist(err)
}

func Copy(src, dest string) {
	util.PanicIfNotNil(cpy.Copy(src, dest, cpy.Options{Sync: true}))
}
