// Package pluginadaptersaresiblings is the check `go/plugin-adapters-are-siblings`, run on the tree with the planted violations that prove it.
package pluginadaptersaresiblings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getSiblingAdapterImport defines the whole rule, so it is tested directly.
func TestWhatTheSiblingAdapterRuleRejects(t *testing.T) {
	own := load.PluginsPrefix + "sample/backend/"
	for _, one := range []struct {
		name, from, dep string
		isRefused       bool
	}{
		{"another adapter of the plugin", "files", own + "internal/adapter/process", true},
		{"a package below another adapter", "files", own + "internal/adapter/engine/onnx", true},
		{"its own package below it", "engine", own + "internal/adapter/engine/onnx", false},
		{"its own root", "engine", own + "internal/adapter/engine", false},
		{"the ports package", "files", own + "ports", false},
		{"the domain", "files", own + "domain/card", false},
		{"an adapter of another plugin", "files", load.PluginsPrefix + "other/backend/internal/adapter/process", false},
		{"the plugin SDK", "files", load.SdkModule + "/vault", false},
	} {
		if got := getSiblingAdapterImport("sample", one.from, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s: refused is %v, want %v", one.name, got, one.isRefused)
		}
	}
}

func TestAdapterNameIsTakenFromThePath(t *testing.T) {
	for path, want := range map[string]string{
		"plugins/sample/backend/internal/adapter/files/files.go":     "files",
		"plugins/sample/backend/internal/adapter/engine/onnx/run.go": "engine",
		"plugins/sample/backend/service/service.go":                  "",
		"plugins/other/backend/internal/adapter/files/files.go":      "",
		"/work/numen/plugins/sample/backend/internal/adapter/a/b.go": "a",
	} {
		if got := getAdapterName("sample", path); got != want {
			t.Errorf("%s: adapter %q, want %q", path, got, want)
		}
	}
}

// An internal/adapter/<a> package of a plugin imports no internal/adapter/<b> of the same plugin: the
// service composition root wires adapters together, and what two adapters share is a port type or a domain type.
func TestNoPluginAdapterImportsASiblingAdapter(t *testing.T) {
	adapters := 0
	for _, dir := range load.ListPluginModules(t) {
		name := filepath.Base(dir)
		imports, err := load.GetImports(dir)
		if err != nil {
			t.Fatal(err)
		}
		for path, deps := range imports {
			from := getAdapterName(name, path)
			if from == "" {
				continue
			}
			adapters++
			for _, dep := range deps {
				if to := getSiblingAdapterImport(name, from, dep); to != "" {
					t.Errorf("%s imports %s: adapter %s reaches adapter %s of the same plugin", path, dep, from, to)
				}
			}
		}
	}
	if adapters == 0 {
		t.Fatal("no plugin adapter source found: a rule checked against nothing")
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
