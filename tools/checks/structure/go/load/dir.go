// Package load holds what the structural checks of the Go code share: the type-checked workspace, the
// findings a rule reports and the layout of the kernel.
package load

import (
	"os"
	"path/filepath"
	"runtime"
)

// ContainerDir is the directory every relative path of a check is written against: the container of the
// kernel, where the checks lived before they moved. Importing this package changes the working directory
// of the test binary to it.
var ContainerDir string

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("load: the location of the package is unknown")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	ContainerDir = filepath.Join(root, "libs", "core", "kernel", "container")
	if err := os.Chdir(ContainerDir); err != nil {
		panic(err)
	}
}
