package direction

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func listOutward() []string {
	return []string{
		load.Module + "adapter", load.Module + "internal/adapter", load.Module + "container", load.Wire,
	}
}

func getRefusedImports(path string) (forbidden []string, named bool) {
	target := strings.TrimPrefix(path, load.Module)
	switch {
	case load.IsUnder(target, "container"):
		return []string{load.Wire}, true
	case strings.HasPrefix(target, "adapter/"),
		strings.HasPrefix(target, "internal/adapter/"),
		strings.HasPrefix(target, "internal/test"),
		load.IsShared(target):
		return nil, false
	}
	return listOutward(), false
}

func isEdgeRefused(pkg, dep string) bool {
	isRefused, _ := getRefusedImports(load.Module + pkg)
	for _, blocked := range isRefused {
		if strings.HasPrefix(dep, blocked) {
			return true
		}
	}
	return false
}

func inward(t *testing.T, goos, goarch string) {
	t.Helper()
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./...")
	listing.Dir = ".."
	listing.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	items, err := listing.Output()
	if err != nil {
		if ran, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	packages := 0
	decoder := json.NewDecoder(bytes.NewReader(items))
	for {
		var pkg struct {
			ImportPath string   `json:"ImportPath"`
			Imports    []string `json:"Imports"`
			Deps       []string `json:"Deps"`
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		forbidden, named := getRefusedImports(pkg.ImportPath)
		if forbidden == nil {
			continue
		}
		packages++
		reaching := pkg.Deps
		if named {
			reaching = pkg.Imports
		}
		target := strings.TrimPrefix(pkg.ImportPath, load.Module)
		for _, dep := range reaching {
			for _, blocked := range forbidden {
				if strings.HasPrefix(dep, blocked) {
					t.Errorf("%s reaches %s", target, strings.TrimPrefix(dep, load.Module))
				}
			}
		}
	}

	// A listing of nothing is a rule checked against nothing.
	if packages == 0 {
		t.Fatal("go list found no package of the core")
	}
}
