// Package load holds what the structural checks of the Go code share: the type-checked workspace, the
// findings a rule reports and the layout of the kernel.
package load

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ContainerDir is the directory every relative path of a check is written against: the container of the
// kernel, where the checks lived before they moved.
var ContainerDir string

// Main changes the working directory of a check's test binary to ContainerDir and runs its tests. A check
// calls it from TestMain.
func Main(m *testing.M) int {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "load: the location of the package is unknown")
		return 1
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	ContainerDir = filepath.Join(root, "libs", "core", "kernel", "container")
	if err := os.Chdir(ContainerDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return m.Run()
}
