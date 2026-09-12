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

// tools is the endpoint an agent calls. It is a driving adapter of the core,
// compiled into a window, and the terminal is not one.
const tools = core + "adapter/mcp"

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

// An agent reaches the vault through tools, and the tools are an endpoint a
// window listens on. The command line reaches the same use cases directly, and
// nothing of the tool surface is linked into it.
//
// What the build resolved is what is read, so the tools reached through another
// package count the same as the tools named outright.
func TestNothingOfTheToolsIsLinkedIntoTheTerminal(t *testing.T) {
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./cmd/...")
	listing.Dir = "../../"
	listed, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	var binaries []string
	var serving []string
	decoder := json.NewDecoder(bytes.NewReader(listed))
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
		if !isLinked(pkg.Deps, tools) {
			continue
		}
		serving = append(serving, pkg.ImportPath)
		if pkg.ImportPath == terminal {
			t.Errorf("%s is compiled from %s: an agent reaches the vault through "+
				"tools, and the tools are a window's endpoint",
				strings.TrimPrefix(pkg.ImportPath, apps), strings.TrimPrefix(tools, core))
		}
	}

	// A listing that did not find the terminal is a rule checked against
	// nothing, and it passes.
	if !slices.Contains(binaries, terminal) {
		t.Fatalf("go list found no %s among %v", terminal, binaries)
	}

	// A listing that finds the tools in no binary is reading a package under
	// another name, and every rule over that name has quietly gone green.
	if len(serving) == 0 {
		t.Fatalf("no binary is compiled from %s: the listing is not reading it", tools)
	}
}

// linking is the whole of the rule above, so it is asked directly. A rule that
// has never been seen to refuse is an assumption.
func TestWhatTheToolsRuleRefuses(t *testing.T) {
	deps := []string{
		core + "adapter/cli",
		core + "adapter/mcpserver",
		core + "usecase/note",
	}
	for _, one := range []struct {
		name    string
		deps    []string
		pkg     string
		refused bool
	}{
		{"the tools outright", append(slices.Clone(deps), tools), tools, true},
		{"the tools through a subpackage", append(slices.Clone(deps), tools+"/serve"), tools, true},
		{"a package the tools' name is a prefix of", deps, tools, false},
		{"a listing of nothing", nil, tools, false},
	} {
		t.Run(one.name, func(t *testing.T) {
			if isLinked(one.deps, one.pkg) != one.refused {
				t.Errorf("linking said %v, want %v", !one.refused, one.refused)
			}
		})
	}
}
