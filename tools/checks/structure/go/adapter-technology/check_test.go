package adaptertechnology

import (
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
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

// A package under adapter/ or internal/adapter/ that calls no use case is a driven adapter and imports a
// technology; one that holds callbacks and no technology is a seam written as an adapter. A package of
// internal/adapter/ that calls a use case is a driving adapter in the wrong folder.
func TestTechnologyRule_RejectsAnAdapterOfCallbacks(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go":          "package find\n\nfunc Execute() {}\n",
		"adapter/window/window.go":      "package window\n\nimport \"fixture/usecase/find\"\n\nfunc Show() { find.Execute() }\n",
		"adapter/disk/disk.go":          "package disk\n\nimport \"os\"\n\nfunc Read() ([]byte, error) { return os.ReadFile(\"x\") }\n",
		"adapter/hook/hook.go":          "package hook\n\ntype Hook struct{ OnChange func() }\n",
		"internal/adapter/host/host.go": "package host\n\nimport \"fixture/usecase/find\"\n\nfunc Serve() { find.Execute() }\n",
	})
	found, driven := getAdapterTechnologyFindings(load.LoadFixture(t, dir), "fixture", map[string]string{})
	want := []string{
		"adapter/hook is a driven adapter that imports no technology: it holds callbacks, not an integration",
		"internal/adapter/host calls a use case: a driving adapter lives under adapter/",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects %v, want %v", found, want)
	}
	if driven != 2 {
		t.Errorf("%d driven adapters read, want 2", driven)
	}
}

func TestDrivenAdaptersImportATechnology(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkDrivenAdaptersImportATechnology(t, load.Workspace(t, "libs/core/kernel"))
}
