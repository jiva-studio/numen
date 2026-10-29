package layers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// plugins is the plugins module path. Plugins are independent and never linked into desktop binaries.
const plugins = "github.com/jiva-studio/numen/plugins"

// terminal is the binary the command line is served through.
const terminal = apps + "desktop/cmd/numen-cli"

// isLinked says whether a listing was compiled from a package, reading a dep as
// the package itself or as something under it. A prefix on its own would take
// adapter/mcpsomething for the tools and a rule that refuses more than it was
// given is as wrong as one that refuses less.
func isLinked(deps []string, pkg string) bool {
	for _, dep := range deps {
		if dep == pkg || strings.HasPrefix(dep, pkg+"/") {
			return true
		}
	}
	return false
}

// Plugins are independent extension units. Nothing of the plugin surface is linked
// directly into the desktop binaries.
func TestNothingOfThePluginsIsLinkedIntoDesktopBinaries(t *testing.T) {
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./cmd/...")
	listing.Dir = "../../"
	items, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	var binaries []string
	decoder := json.NewDecoder(bytes.NewReader(items))
	for {
		var pkg struct {
			ImportPath string
			// Deps is everything the package is compiled from, at any remove.
			Deps []string
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		binaries = append(binaries, pkg.ImportPath)
		if isLinked(pkg.Deps, plugins) {
			t.Errorf("%s is compiled from %s: plugins must not be linked into desktop binaries",
				pkg.ImportPath, plugins)
		}
	}

	// A listing that did not find the terminal is a rule checked against
	// nothing, and it passes.
	if !slices.Contains(binaries, terminal) {
		t.Fatalf("go list found no %s among %v", terminal, binaries)
	}
}

// linking is the whole of the rule above, so it is asked directly. A rule that
// has never been seen to refuse is an assumption.
func TestPluginsRule_RejectsPluginsDependency(t *testing.T) {
	deps := []string{
		core + "adapter/cli",
		core + "usecase/note",
	}
	for _, one := range []struct {
		name      string
		deps      []string
		pkg       string
		isRefused bool
	}{
		{"plugins outright", append(slices.Clone(deps), plugins), plugins, true},
		{"plugins subpackage", append(slices.Clone(deps), plugins+"/agents.mcp.server"), plugins, true},
		{"a package the plugins' name is a prefix of", deps, plugins, false},
		{"a listing of nothing", nil, plugins, false},
	} {
		t.Run(one.name, func(t *testing.T) {
			if isLinked(one.deps, one.pkg) != one.isRefused {
				t.Errorf("linking said %v, want %v", !one.isRefused, one.isRefused)
			}
		})
	}
}
