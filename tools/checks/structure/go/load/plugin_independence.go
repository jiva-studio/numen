package load

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// NumenRoot is what every module of this repository is named under.
const NumenRoot = "github.com/jiva-studio/numen/"

// NotesModule is the module of the note model, which plugins do not name.
const NotesModule = NumenRoot + "libs/note"

// RepoRootDir is the repository root, relative to this package.
const RepoRootDir = "../../../.."

func IsWithin(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

func GetPluginDependencyRefusal(mod, dep string) string {
	switch {
	case IsWithin(dep, mod), IsSDK(dep):
		return ""
	case IsWithin(dep, "connectrpc.com/connect"), IsWithin(dep, "google.golang.org/protobuf"):
		return "a plugin calls and answers the core through the plugin SDK, not through the RPC library"
	case !strings.HasPrefix(dep, NumenRoot):
		return ""
	case IsWithin(dep, NotesModule):
		return "a plugin names no package of the notes library; it reaches notes through the plugin SDK"
	case IsPlugin(dep):
		return "a plugin names no other plugin"
	case IsWithin(dep, strings.TrimSuffix(Module, "/")):
		return "a plugin names no package of the core"
	}
	return "a plugin names only the plugin SDK and third-party code"
}

func ListPluginModules(t *testing.T) []string {
	t.Helper()
	dirs, err := GetPluginModules(RepoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	return dirs
}

// GetPluginModules lists the folders under root/plugins that hold a go.mod. It fails when there is none.
func GetPluginModules(root string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(root, "plugins", "*", "go.mod"))
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no plugin module under %s/plugins: a rule checked against nothing", root)
	}
	dirs := make([]string, len(found))
	for i, gomod := range found {
		dirs[i] = filepath.Dir(gomod)
	}
	return dirs, nil
}

func GetImports(root string) (map[string][]string, error) {
	imports := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		deps, err := readImports(fset, path)
		if len(deps) > 0 {
			imports[path] = append(imports[path], deps...)
		}
		return err
	})
	return imports, err
}

func readImports(fset *token.FileSet, path string) ([]string, error) {
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var deps []string
	for _, spec := range file.Imports {
		dep, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return deps, err
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// DepguardRule is one rule of the depguard linter in a golangci configuration.
type DepguardRule struct {
	Files []string `yaml:"files"`
	Allow []string `yaml:"allow"`
	Deny  []struct {
		Pkg string `yaml:"pkg"`
	} `yaml:"deny"`
}
