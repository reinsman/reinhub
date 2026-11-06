package testutil

import (
	"os"
	"path"
	"runtime"
)

// 设置当前工作目录为正常二进制文件所在目录
/*
在 所需固定工作目录的testing文件加上

	_ "<your-module>/internal/testutil"

*/
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..", "..", "bin")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}
