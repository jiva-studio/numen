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

// Main changes the working directory of a check's test binary to the container of the kernel, where the checks
// lived before they moved and against which their relative paths are written, and runs its tests. A check
// calls it from TestMain.
func Main(m *testing.M) int {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "load: the location of the package is unknown")
		return 1
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	if err := os.Chdir(filepath.Join(root, "libs", "core", "kernel", "container")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return m.Run()
}
