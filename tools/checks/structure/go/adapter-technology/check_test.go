// Package adaptertechnology is the check `go/adapter-technology`, run on the tree with the planted violations that prove it.
package adaptertechnology

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A technology package gives access to something outside the process: the file system, the network, a child
// process, a database, the operating system, a device or a vendor SDK. The standard time package is not one,
// because reading the clock is a technology only for the clock adapter, which the exceptions list names.
// Third-party packages count, except the ones that only compute.
func TestTechnologyImportRule_NamesWhatReachesOutsideTheProcess(t *testing.T) {
	for path, want := range map[string]bool{
		"os": true, "os/exec": true, "io/fs": true, "net/http": true, "net": true, "database/sql": true,
		"golang.org/x/sys/unix": true, "github.com/rjeczalik/notify": true, "modernc.org/sqlite": true,
		"time": false, "io": false, "context": false, "sync": false, "encoding/json": false, "path/filepath": false,
		"golang.org/x/sync/errgroup": false, "github.com/jiva-studio/numen/libs/note/domain": false,
	} {
		if got := load.IsTechnologyImport(path); got != want {
			t.Errorf("%s is a technology: %v, want %v", path, got, want)
		}
	}
}

// A package under adapter/ or internal/adapter/ that calls no use case is a driven adapter and calls a
// technology, a function or a method of a package that reaches outside the process; one that holds
// callbacks, or imports a technology package only for its types, is a seam written as an adapter. A
// package of internal/adapter/ that calls a use case is a driving adapter in the wrong folder.
func TestTechnologyRule_RejectsAnAdapterThatCallsNoTechnology(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go":          "package find\n\nfunc Execute() {}\n",
		"adapter/window/window.go":      "package window\n\nimport \"fixture/usecase/find\"\n\nfunc Show() { find.Execute() }\n",
		"adapter/disk/disk.go":          "package disk\n\nimport \"os\"\n\nfunc Read() ([]byte, error) { return os.ReadFile(\"x\") }\n",
		"adapter/file/file.go":          "package file\n\nimport \"os\"\n\nfunc Close(f *os.File) error { return f.Close() }\n",
		"adapter/hook/hook.go":          "package hook\n\ntype Hook struct{ OnChange func() }\n",
		"adapter/typeonly/typeonly.go":  "package typeonly\n\nimport \"net/http\"\n\nfunc New() http.Handler { return nil }\n\nvar _ = http.HandlerFunc(nil)\n",
		"adapter/convert/convert.go":    "package convert\n\nimport \"os\"\n\ntype Mode = os.FileMode\n\nfunc Wrap(m uint32) Mode { return os.FileMode(m) }\n",
		"internal/adapter/host/host.go": "package host\n\nimport \"fixture/usecase/find\"\n\nfunc Serve() { find.Execute() }\n",
	})
	found, driven := getAdapterTechnologyFindings(load.LoadFixture(t, dir), "fixture", map[string]string{})
	want := []string{
		"adapter/convert is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/hook is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/typeonly is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"internal/adapter/host calls a use case: a driving adapter lives under adapter/",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects %v, want %v", found, want)
	}
	if driven != 5 {
		t.Errorf("%d driven adapters read, want 5", driven)
	}
}

func TestDrivenAdaptersCallATechnology(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkDrivenAdaptersCallATechnology(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}

// The libraries that frame the wire are no technology a call reaches outside the process through.
func TestTechnologyCallRule_ExcludesTheWireFraming(t *testing.T) {
	for path, want := range map[string]bool{
		"connectrpc.com/connect": false, "google.golang.org/protobuf/proto": false,
		"os": true, "net/http": true, "modernc.org/sqlite": true, "time": false,
	} {
		if got := isTechnologyCall(path); got != want {
			t.Errorf("a call into %s reaches outside the process: %v, want %v", path, got, want)
		}
	}
}
