package osutil

import (
	"os"
	"path/filepath"
)

// 获取当前可执行文件所在目录
func GetBinFolder() string {
	exePath, _ := os.Executable()
	return filepath.Dir(exePath)
}
