package machineaccess

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getMachinePackages() map[string]bool {
	return map[string]bool{
		"os": true, "os/exec": true, "net": true, "net/http": true, "syscall": true,
	}
}

// golang.org/x/sys is the kernel under another name: every package beneath it
// is one operating system's own calls and error numbers.
const systemCalls = "golang.org/x/sys/"

func isMachinePackage(to string) bool {
	return getMachinePackages()[to] || strings.HasPrefix(to, systemCalls)
}

func getFilesystemCalls() map[string]bool {
	return map[string]bool{
		"EvalSymlinks": true, "Glob": true, "Walk": true, "WalkDir": true, "Abs": true,
	}
}
