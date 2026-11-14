package load

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"
)

// WorkspacePackages is the type-checked packages of the modules of the workspace, tests included, by
// the directory of the module. The rules that walk it share one load and never change it.
type WorkspacePackages map[string][]*packages.Package

func (w WorkspacePackages) ListModule(t *testing.T, dir string) []*packages.Package {
	t.Helper()
	pkgs, ok := w[dir]
	if !ok {
		t.Fatalf("%s is not a module of the workspace", dir)
	}
	return pkgs
}

// Workspace loads and type-checks the modules of the workspace named by their path from the repository
// root, or every module when none is named.
func Workspace(t *testing.T, modules ...string) WorkspacePackages {
	t.Helper()
	return loadWorkspace(t, nil, modules)
}

// LoadWorkspaceFor loads and type-checks the modules as Workspace does, built for platform.
func LoadWorkspaceFor(t *testing.T, platform Platform, modules ...string) WorkspacePackages {
	t.Helper()
	return loadWorkspace(t, platform.GetEnv(), modules)
}

func loadWorkspace(t *testing.T, env []string, modules []string) WorkspacePackages {
	t.Helper()
	dirs := ListWorkspaceModules(t)
	if len(modules) > 0 {
		dirs = nil
		for _, rel := range modules {
			dirs = append(dirs, absoluteDir(t, rel))
		}
	}
	var (
		group   sync.WaitGroup
		mu      sync.Mutex
		byDir   = make(WorkspacePackages, len(dirs))
		failure error
	)
	slots := make(chan struct{}, max(1, runtime.GOMAXPROCS(0)/2))
	for _, dir := range dirs {
		group.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			pkgs, err := packages.Load(NewTypedConfig(dir, env), "./...")
			mu.Lock()
			defer mu.Unlock()
			if err != nil && failure == nil {
				failure = err
			}
			byDir[dir] = pkgs
		})
	}
	group.Wait()
	if failure != nil {
		t.Fatal(failure)
	}
	return byDir
}

// IsTestVariant reports whether pkg is a package compiled with its tests.
func IsTestVariant(pkg *packages.Package) bool {
	return strings.Contains(pkg.ID, " [") || strings.HasSuffix(pkg.ID, ".test")
}

func absoluteDir(t *testing.T, rel string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(RepoRootDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
