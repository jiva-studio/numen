package container

import (
	"bufio"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// numenModules is what every module of this repository is named under.
const numenModules = "github.com/jiva-studio/numen/modules/"

// pluginContract are the modules a plugin may name besides its own.
var pluginContract = []string{
	numenModules + "libs/plugin-sdk-go",
	numenModules + "libs/protocol",
}

// getPluginRefusal says why a plugin module may not name dep, or "" when it may.
// A plugin names its own module, the plugin contract and third-party code.
func getPluginRefusal(plugin, dep string) string {
	if !strings.HasPrefix(dep, numenModules) {
		return ""
	}
	own := numenModules + "plugins/" + plugin
	if dep == own || strings.HasPrefix(dep, own+"/") {
		return ""
	}
	for _, allowed := range pluginContract {
		if dep == allowed || strings.HasPrefix(dep, allowed+"/") {
			return ""
		}
	}
	if strings.HasPrefix(dep, numenModules+"plugins/") {
		return "a plugin names no other plugin"
	}
	return "a plugin names the plugin contract and no other module of the repository"
}

// What getPluginRefusal says is the whole of the rule, so it is asked directly.
func TestWhatThePluginRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		plugin, dep string
		isRefused   bool
	}{
		{"ocr", numenModules + "libs/core/domain", true},
		{"ocr", numenModules + "libs/core", true},
		{"ocr", numenModules + "libs/ui", true},
		{"ocr", numenModules + "plugins/embeddings/backend/engine", true},
		{"agent", numenModules + "plugins/agent-acp/backend", true},
		{"agent-acp", numenModules + "plugins/agent", true},
		{"ocr", numenModules + "plugins/ocr/backend/engine", false},
		{"ocr", numenModules + "libs/plugin-sdk-go/server", false},
		{"ocr", numenModules + "libs/protocol/gen/numen/v1", false},
		{"ocr", "github.com/connectrpc/connect", false},
		{"ocr", "net/http", false},
	} {
		if got := getPluginRefusal(one.plugin, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s names %s: refused is %v, want %v", one.plugin, one.dep, got, one.isRefused)
		}
	}
}

// A plugin module requires and imports only its own packages, the plugin
// contract and third-party code.
func TestNoPluginNamesAnotherModuleOfTheRepository(t *testing.T) {
	root := filepath.Join("..", "..", "..", "plugins")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	plugins := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		plugin := entry.Name()
		dir := filepath.Join(root, plugin)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			continue
		}
		plugins++
		for _, dep := range readRequires(t, filepath.Join(dir, "go.mod")) {
			if why := getPluginRefusal(plugin, dep); why != "" {
				t.Errorf("%s requires %s: %s", plugin, dep, why)
			}
		}
		err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				if e.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, one := range file.Imports {
				dep, err := strconv.Unquote(one.Path.Value)
				if err != nil {
					continue
				}
				if why := getPluginRefusal(plugin, dep); why != "" {
					t.Errorf("%s imports %s: %s", path, dep, why)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// A listing of nothing is a rule checked against nothing.
	if plugins == 0 {
		t.Fatal("no plugin module found")
	}
}

// readRequires returns the module paths a go.mod requires.
func readRequires(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var deps []string
	inBlock := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "require (":
			inBlock = true
		case line == ")":
			inBlock = false
		case strings.HasPrefix(line, "require "):
			deps = append(deps, strings.Fields(strings.TrimPrefix(line, "require "))[0])
		case inBlock && line != "" && !strings.HasPrefix(line, "//"):
			deps = append(deps, strings.Fields(line)[0])
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return deps
}
