package load

import (
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
)

func IsTechnologyImport(path string) bool {
	switch {
	case path == "os", path == "io/fs", path == "syscall", path == "database/sql", path == "embed", path == "plugin", path == "net":
		return true
	case strings.HasPrefix(path, "os/"), strings.HasPrefix(path, "net/"), strings.HasPrefix(path, "golang.org/x/sys/"):
		return true
	case strings.HasPrefix(path, "github.com/jiva-studio/numen/"):
		return false
	}
	first, _, _ := strings.Cut(path, "/")
	if !strings.Contains(first, ".") {
		return false
	}
	for _, pure := range []string{"golang.org/x/sync", "golang.org/x/text", "golang.org/x/exp"} {
		if IsWithin(path, pure) {
			return false
		}
	}
	return true
}
