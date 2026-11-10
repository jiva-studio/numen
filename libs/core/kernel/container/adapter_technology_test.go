package container

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// isTechnologyImport tells whether a package gives access to something outside the process: the file
// system, the network, a child process, a database, the operating system, a device, or a vendor SDK. The
// standard time package is not one: reading the clock is the technology only for the clock adapter, which
// the helper list names. Third-party packages count, except the ones that only compute.
func isTechnologyImport(path string) bool {
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
		if isWithin(path, pure) {
			return false
		}
	}
	return true
}

// getTechnologyFreeAdapters names the adapter packages that reach no technology and are none of the
// adapters a layer rule already places, each with the reason it needs no technology.
func getTechnologyFreeAdapters() map[string]string {
	return map[string]string{
		"internal/adapter/clock": "the machine clock itself, read through the time package",
	}
}

func isUseCaseImport(root, path string) bool {
	return isWithin(path, root+"/usecase")
}

// getAdapterTechnologyFindings reports the adapter packages that call a use case from a place a driving
// adapter does not belong, and the driven adapters that import no technology.
func getAdapterTechnologyFindings(pkgs []*packages.Package, root string, free map[string]string) (found []string, driven int) {
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		isInternalAdapter := isWithin(rel, "internal/adapter")
		if !isInternalAdapter && !isWithin(rel, "adapter") {
			continue
		}
		isDriving := false
		isTechnical := false
		for path := range pkg.Imports {
			isDriving = isDriving || isUseCaseImport(root, path)
			isTechnical = isTechnical || isTechnologyImport(path)
		}
		switch {
		case isDriving && isInternalAdapter:
			found = append(found, rel+" calls a use case: a driving adapter lives under adapter/")
		case isDriving:
		default:
			driven++
			if _, isFree := free[rel]; !isTechnical && !isFree {
				found = append(found, rel+" is a driven adapter that imports no technology: it holds callbacks, not an integration")
			}
		}
	}
	slices.Sort(found)
	return found, driven
}

func checkDrivenAdaptersImportATechnology(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkgs := getModulePackages(t, workspace, "libs/core/kernel")
	free := getTechnologyFreeAdapters()
	found, driven := getAdapterTechnologyFindings(pkgs, coreImportPath, free)
	for _, one := range found {
		t.Error(one)
	}
	for name, reason := range free {
		if !slices.ContainsFunc(pkgs, func(pkg *packages.Package) bool { return pkg.PkgPath == coreImportPath+"/"+name }) || reason == "" {
			t.Errorf("the entry %s names no package of the kernel or has no reason", name)
		}
	}
	if driven < 20 {
		t.Fatalf("%d driven adapters read: the rule is checked against nothing", driven)
	}
}

func TestTechnologyImportRule_NamesWhatReachesOutsideTheProcess(t *testing.T) {
	for path, want := range map[string]bool{
		"os": true, "os/exec": true, "io/fs": true, "net/http": true, "net": true, "database/sql": true,
		"golang.org/x/sys/unix": true, "github.com/rjeczalik/notify": true, "modernc.org/sqlite": true,
		"time": false, "io": false, "context": false, "sync": false, "encoding/json": false, "path/filepath": false,
		"golang.org/x/sync/errgroup": false, "github.com/jiva-studio/numen/libs/note/domain": false,
	} {
		if got := isTechnologyImport(path); got != want {
			t.Errorf("%s is a technology: %v, want %v", path, got, want)
		}
	}
}

func TestTechnologyRule_RejectsAnAdapterOfCallbacks(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"usecase/find/find.go":          "package find\n\nfunc Execute() {}\n",
		"adapter/window/window.go":      "package window\n\nimport \"fixture/usecase/find\"\n\nfunc Show() { find.Execute() }\n",
		"adapter/disk/disk.go":          "package disk\n\nimport \"os\"\n\nfunc Read() ([]byte, error) { return os.ReadFile(\"x\") }\n",
		"adapter/hook/hook.go":          "package hook\n\ntype Hook struct{ OnChange func() }\n",
		"internal/adapter/host/host.go": "package host\n\nimport \"fixture/usecase/find\"\n\nfunc Serve() { find.Execute() }\n",
	})
	found, driven := getAdapterTechnologyFindings(loadFixture(t, dir), "fixture", map[string]string{})
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
