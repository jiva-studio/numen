package pluginboundary

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/mod/modfile"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getPluginRequirements(file *modfile.File) []string {
	var named []string
	for _, required := range file.Require {
		if load.IsPlugin(required.Mod.Path) {
			named = append(named, "requires "+required.Mod.Path)
		}
	}
	for _, replaced := range file.Replace {
		isPluginFolder := replaced.New.Version == "" &&
			slices.Contains(strings.Split(filepath.ToSlash(replaced.New.Path), "/"), "plugins")
		if load.IsPlugin(replaced.Old.Path) || load.IsPlugin(replaced.New.Path) || isPluginFolder {
			named = append(named, "replaces "+replaced.Old.Path+" with "+replaced.New.Path)
		}
	}
	return named
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
