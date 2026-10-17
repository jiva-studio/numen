package container

import (
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
		text, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		for _, why := range getGoModRefusals(plugin, parseGoMod(string(text))) {
			t.Errorf("%s %s", plugin, why)
		}
		err = filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
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

// goMod is what a go.mod says about the modules it names: the ones it
// requires, and the local directories it substitutes for a module.
type goMod struct {
	requires []string
	// replaced maps a module path to the local directory it is built from.
	replaced map[string]string
}

// parseGoMod reads the require and replace directives of a go.mod, in single
// lines and in blocks, with trailing comments removed.
func parseGoMod(text string) goMod {
	mod := goMod{replaced: map[string]string{}}
	block := ""
	for _, line := range strings.Split(text, "\n") {
		line, _, _ = strings.Cut(line, "//")
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		switch {
		case len(fields) == 0:
		case line == ")":
			block = ""
		case len(fields) == 2 && fields[1] == "(":
			block = fields[0]
		case block != "":
			mod.read(block, fields)
		default:
			mod.read(fields[0], fields[1:])
		}
	}
	return mod
}

// read records one directive of the given kind.
func (mod *goMod) read(kind string, fields []string) {
	switch kind {
	case "require":
		if len(fields) > 0 {
			mod.requires = append(mod.requires, fields[0])
		}
	case "replace":
		for i, field := range fields {
			if field == "=>" && i+1 < len(fields) && strings.HasPrefix(fields[i+1], ".") {
				mod.replaced[fields[0]] = fields[i+1]
			}
		}
	}
}

// getGoModRefusals says why a plugin's go.mod may not stand: it requires a
// module of the repository the plugin may not name, or builds any module from
// a local directory other than the one the repository's own module is at.
func getGoModRefusals(plugin string, mod goMod) []string {
	var refusals []string
	for _, dep := range mod.requires {
		if why := getPluginRefusal(plugin, dep); why != "" {
			refusals = append(refusals, "requires "+dep+": "+why)
		}
	}
	for from, to := range mod.replaced {
		if why := getPluginRefusal(plugin, from); why != "" {
			refusals = append(refusals, "replaces "+from+" with "+to+": "+why)
		} else if !strings.HasPrefix(from, numenModules) {
			refusals = append(refusals, "replaces "+from+" with "+to+": third-party code is built from its own module")
		}
	}
	return refusals
}

func TestWhatThePluginGoModRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		name, text string
		refused    int
	}{
		{"a core requirement in a block", "require (\n\t" + numenModules + "libs/core v0.0.0\n)", 1},
		{"a block opener with a comment", "require ( // deps\n\t" + numenModules + "plugins/ocr v0.0.0\n)", 1},
		{"a single-line requirement", "require " + numenModules + "plugins/ocr v0.0.0", 1},
		{"a third-party module built from core", "require example.com/x v1.0.0\nreplace example.com/x => ../../libs/core", 1},
		{"the contract", "require (\n\t" + numenModules + "libs/protocol v0.0.0\n\texample.com/x v1.0.0\n)\nreplace " + numenModules + "libs/protocol => ../../libs/protocol", 0},
	} {
		if got := len(getGoModRefusals("agent", parseGoMod(one.text))); got != one.refused {
			t.Errorf("%s: %d refusals, want %d", one.name, got, one.refused)
		}
	}
}
