package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// outward is what the core may not be compiled from: an adapter, and the place
// adapters are assembled.
var outward = []string{module + "internal/adapter", module + "internal/container"}

// Imports point inward. What the build resolved is what is read, so a package
// the core reaches through another answers for it too.
//
// A test file is left out: a test stands outside the package it exercises and
// builds the adapters that stand in for the real ones.
func TestNothingTheCoreIsCompiledFromReachesOutward(t *testing.T) {
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./internal/core/...")
	listing.Dir = filepath.Join("..", "..")
	listed, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	packages := 0
	decoder := json.NewDecoder(bytes.NewReader(listed))
	for {
		var pkg struct {
			ImportPath string
			// Deps is everything this package is compiled from, at whatever
			// remove.
			Deps []string
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		packages++
		for _, dep := range pkg.Deps {
			for _, refused := range outward {
				if strings.HasPrefix(dep, refused) {
					t.Errorf("%s reaches %s", strings.TrimPrefix(pkg.ImportPath, module),
						strings.TrimPrefix(dep, module))
				}
			}
		}
	}

	// A listing of nothing is a rule checked against nothing.
	if packages == 0 {
		t.Fatal("go list found no package under internal/core")
	}
}
