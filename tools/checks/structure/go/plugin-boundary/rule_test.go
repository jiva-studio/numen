package pluginboundary

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/mod/modfile"
)

const linkedRegistration = "RegisterBuiltIn"

type pluginRequirement struct {
	module string
	line   string
}

func getPluginRequirements(file *modfile.File) []pluginRequirement {
	var named []pluginRequirement
	for _, required := range file.Require {
		if load.IsPlugin(required.Mod.Path) {
			named = append(named, pluginRequirement{required.Mod.Path, "requires " + required.Mod.Path})
		}
	}
	for _, replaced := range file.Replace {
		isPluginFolder := replaced.New.Version == "" &&
			slices.Contains(strings.Split(filepath.ToSlash(replaced.New.Path), "/"), "plugins")
		if load.IsPlugin(replaced.Old.Path) || load.IsPlugin(replaced.New.Path) || isPluginFolder {
			named = append(named, pluginRequirement{replaced.Old.Path, "replaces " + replaced.Old.Path + " with " + replaced.New.Path})
		}
	}
	return named
}

func isRegistered(module string, registered []string) bool {
	return slices.ContainsFunc(registered, func(imported string) bool {
		return imported == module || strings.HasPrefix(imported, module+"/")
	})
}

func listRegisteredPlugins(t *testing.T, dir string) []string {
	t.Helper()
	var registered []string
	err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if file == dir {
				return nil
			}
			if name := entry.Name(); name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(file, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		registered = append(registered, listRegisteredImports(parsed)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return registered
}

func listRegisteredImports(file *ast.File) []string {
	plugins := map[string]string{}
	for _, imported := range file.Imports {
		importPath, err := strconv.Unquote(imported.Path.Value)
		if err != nil || !load.IsPlugin(importPath) {
			continue
		}
		name := path.Base(importPath)
		if imported.Name != nil {
			name = imported.Name.Name
		}
		plugins[name] = importPath
	}
	var registered []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if callee, ok := call.Fun.(*ast.SelectorExpr); !ok || callee.Sel.Name != linkedRegistration {
			return true
		}
		for _, arg := range call.Args {
			ast.Inspect(arg, func(node ast.Node) bool {
				if selector, ok := node.(*ast.SelectorExpr); ok {
					if pkg, ok := selector.X.(*ast.Ident); ok && plugins[pkg.Name] != "" {
						registered = append(registered, plugins[pkg.Name])
					}
				}
				return true
			})
		}
		return true
	})
	return registered
}

func parseGoMod(t *testing.T, text string) *modfile.File {
	t.Helper()
	file, err := modfile.Parse("go.mod", []byte(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func listPackages(t *testing.T, dir, goos, goarch string) []listed {
	t.Helper()
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./...")
	listing.Dir = dir
	listing.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	out, err := listing.Output()
	if err != nil {
		if ran, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}
	var packages []listed
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var pkg listed
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

// listed is a package as `go list` reports it, with all transitive dependencies
// it is compiled from.
type listed struct {
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
	Deps       []string `json:"Deps"`
}
