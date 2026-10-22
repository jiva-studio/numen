package container

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// numenRoot is what every module of this repository is named under.
const numenRoot = "github.com/jiva-studio/numen/"

// notesModule is the module of the note model, which plugins may name.
const notesModule = numenRoot + "modules/libs/notes"

// numenModulesDir is the folder the modules live in, relative to this package.
var numenModulesDir = filepath.Join("..", "..", "..")

// getPluginDependencyRefusal says why the plugin module mod may not name dep, in
// a requirement or in an import, or "" when it may. A plugin names the plugin
// SDK, the protocol and third-party code, and its own packages.
func getPluginDependencyRefusal(mod, dep string) string {
	switch {
	case !strings.HasPrefix(dep, numenRoot), dep == mod, strings.HasPrefix(dep, mod+"/"):
		return ""
	case isSDK(dep), dep == wire, strings.HasPrefix(dep, wire+"/"):
		return ""
	case dep == notesModule, strings.HasPrefix(dep, notesModule+"/"):
		return ""
	case isPlugin(dep):
		return "a plugin names no other plugin"
	case dep == strings.TrimSuffix(module, "/"), strings.HasPrefix(dep, module):
		return "a plugin names no package of the core"
	}
	return "a plugin names only the plugin SDK, the protocol, the notes and third-party code"
}

// What getPluginDependencyRefusal says is the whole of the rule, so it is asked
// directly.
func TestWhatThePluginDependencyRuleRefuses(t *testing.T) {
	self := pluginsPrefix + "sample"
	for _, one := range []struct {
		dep       string
		isRefused bool
	}{
		{pluginsPrefix + "other", true},
		{pluginsPrefix + "other/backend/service", true},
		{module + "port", true},
		{strings.TrimSuffix(module, "/"), true},
		{numenRoot + "modules/apps/desktop/app", true},
		{self, false},
		{self + "/backend/service", false},
		{sdkModule, false},
		{sdkModule + "/vault", false},
		{sdkModule + "/server", false},
		{wire, false},
		{notesModule, false},
		{notesModule + "/domain", false},
		{wire + "/gen/numen/v1", false},
		{"connectrpc.com/connect", false},
		{"github.com/jiva-studio/other", false},
	} {
		if got := getPluginDependencyRefusal(self, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s names %s: failure is %v, want %v", self, one.dep, got, one.isRefused)
		}
	}
}

// listPluginModules is the folder of every plugin that is a Go module, slashed
// and relative to the root of the modules.
func listPluginModules(t *testing.T) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(numenModulesDir, "plugins", "*", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if len(found) < 2 {
		t.Fatalf("%d plugin modules found: a rule checked against nothing", len(found))
	}
	dirs := make([]string, len(found))
	for i, gomod := range found {
		dirs[i] = filepath.Dir(gomod)
	}
	return dirs
}

// A plugin module requires the plugin SDK, the protocol and third-party code.
func TestNoPluginModuleRequiresMoreThanTheSDKAndTheProtocol(t *testing.T) {
	for _, dir := range listPluginModules(t) {
		text, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		mod := getModulePath(string(text))
		if want := pluginsPrefix + filepath.Base(dir); mod != want {
			t.Errorf("%s declares module %q, want %q", dir, mod, want)
		}
		for _, dep := range parseGoMod(string(text)).requires {
			if why := getPluginDependencyRefusal(mod, dep); why != "" {
				t.Errorf("%s requires %s: %s", mod, dep, why)
			}
		}
	}
}

// getImports is every path a Go file in the tree at root imports, by the file
// that imports it. A folder of dependencies is not part of the tree.
func getImports(root string) (map[string][]string, error) {
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
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			dep, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			imports[path] = append(imports[path], dep)
		}
		return nil
	})
	return imports, err
}

// A plugin imports the plugin SDK, the protocol and third-party code, in its
// sources and in its tests.
func TestNoPluginImportsAnotherPluginOrTheCore(t *testing.T) {
	for _, dir := range listPluginModules(t) {
		mod := pluginsPrefix + filepath.Base(dir)
		imports, err := getImports(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(imports) == 0 {
			t.Fatalf("%s imports nothing: a rule checked against nothing", dir)
		}
		for path, deps := range imports {
			for _, dep := range deps {
				if why := getPluginDependencyRefusal(mod, dep); why != "" {
					t.Errorf("%s imports %s: %s", path, dep, why)
				}
			}
		}
	}
}

// depguardRule is one rule of the depguard linter in a golangci configuration.
type depguardRule struct {
	Files []string `yaml:"files"`
	Deny  []struct {
		Pkg string `yaml:"pkg"`
	} `yaml:"deny"`
}

// getDepguardGaps lists what the depguard rules of a golangci configuration
// leave open: each plugin has a rule over its own folder that denies the core
// and every other plugin, as a package and as a folder of packages.
func getDepguardGaps(config []byte, plugins []string) ([]string, error) {
	var parsed struct {
		Linters struct {
			Settings struct {
				Depguard struct {
					Rules map[string]depguardRule `yaml:"rules"`
				} `yaml:"depguard"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(config, &parsed); err != nil {
		return nil, err
	}
	var gaps []string
	for _, plugin := range plugins {
		glob := "**/modules/plugins/" + plugin + "/**"
		var denied []string
		for _, rule := range parsed.Linters.Settings.Depguard.Rules {
			if !slices.Contains(rule.Files, glob) {
				continue
			}
			for _, deny := range rule.Deny {
				denied = append(denied, deny.Pkg)
			}
		}
		if denied == nil {
			gaps = append(gaps, plugin+" has no depguard rule")
			continue
		}
		wanted := []string{strings.TrimSuffix(module, "/")}
		for _, other := range plugins {
			if other != plugin {
				wanted = append(wanted, pluginsPrefix+other+"$", pluginsPrefix+other+"/")
			}
		}
		for _, pkg := range wanted {
			if !slices.Contains(denied, pkg) {
				gaps = append(gaps, plugin+" does not deny "+pkg)
			}
		}
	}
	return gaps, nil
}

// What getDepguardGaps says is the whole of the rule, so it is asked directly.
func TestWhatTheDepguardRulesRefuse(t *testing.T) {
	plugins := []string{"a", "b"}
	complete := `
linters:
  settings:
    depguard:
      rules:
        a:
          files: ["**/modules/plugins/a/**"]
          deny:
            - pkg: "` + strings.TrimSuffix(module, "/") + `"
            - pkg: "` + pluginsPrefix + `b$"
            - pkg: "` + pluginsPrefix + `b/"
        b:
          files: ["**/modules/plugins/b/**"]
          deny:
            - pkg: "` + strings.TrimSuffix(module, "/") + `"
            - pkg: "` + pluginsPrefix + `a$"
            - pkg: "` + pluginsPrefix + `a/"
`
	gaps, err := getDepguardGaps([]byte(complete), plugins)
	if err != nil || len(gaps) != 0 {
		t.Fatalf("a complete configuration has gaps %v (%v)", gaps, err)
	}
	missingRule := strings.Replace(complete, "modules/plugins/b/**", "modules/plugins/c/**", 1)
	missingDeny := strings.Replace(complete, pluginsPrefix+`a/"`, pluginsPrefix+`z/"`, 1)
	for name, config := range map[string]string{
		"a plugin without a rule":         missingRule,
		"a rule that leaves a plugin out": missingDeny,
	} {
		if gaps, err := getDepguardGaps([]byte(config), plugins); err != nil || len(gaps) == 0 {
			t.Errorf("%s is not found (%v)", name, err)
		}
	}
}

// Every plugin module has a depguard rule that denies the core and every other
// plugin.
func TestEveryPluginModuleHasADepguardRuleDenyingTheCoreAndThePlugins(t *testing.T) {
	var plugins []string
	for _, dir := range listPluginModules(t) {
		plugins = append(plugins, filepath.Base(dir))
	}
	config, err := os.ReadFile(filepath.Join(numenModulesDir, "plugins", ".golangci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	gaps, err := getDepguardGaps(config, plugins)
	if err != nil {
		t.Fatal(err)
	}
	for _, gap := range gaps {
		t.Error(gap)
	}
}

// pluginIDLiteral is a plugin id spelled out: the namespace of the plugins that
// ship with numen, and a name after it.
var pluginIDLiteral = regexp.MustCompile(`md\.numen\.[A-Za-z0-9]`)

// getPluginIDLiterals is every string literal of a file that spells out a
// plugin id.
func getPluginIDLiterals(file *ast.File) []string {
	var spelled []string
	ast.Inspect(file, func(n ast.Node) bool {
		if literal, ok := n.(*ast.BasicLit); ok && literal.Kind == token.STRING && pluginIDLiteral.MatchString(literal.Value) {
			spelled = append(spelled, literal.Value)
		}
		return true
	})
	return spelled
}

func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "src.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// What getPluginIDLiterals says is the whole of the rule, so it is asked
// directly.
func TestWhatThePluginIDRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		src       string
		isRefused bool
	}{
		{`package p; var id = "md.numen.sample"`, true},
		{"package p; var id = `md.numen.sample`", true},
		{`package p; var msg = "e.g. md.numen.sample"`, true},
		{`package p; var prefix = "md.numen." + name`, false},
		{`package p; var id = "com.example.sample"`, false},
		{`package p; var path = "github.com/jiva-studio/numen/modules"`, false},
		{`package p; // md.numen.sample`, false},
	} {
		if got := len(getPluginIDLiterals(parseSource(t, one.src))) > 0; got != one.isRefused {
			t.Errorf("%s: wantErr is %v, want %v", one.src, got, one.isRefused)
		}
	}
}

// listSources is every Go file of the tree at root that is not a test, outside
// a folder of dependencies.
func listSources(root string) ([]string, error) {
	var sources []string
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
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			sources = append(sources, path)
		}
		return nil
	})
	return sources, err
}

// parseSources parses every file and hands each parsed file to check.
func parseSources(t *testing.T, roots []string, check func(path string, file *ast.File)) {
	t.Helper()
	fset := token.NewFileSet()
	count := 0
	for _, root := range roots {
		sources, err := listSources(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range sources {
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			count++
			check(path, file)
		}
	}
	if count == 0 {
		t.Fatalf("no Go file found under %v: a rule checked against nothing", roots)
	}
}

// The core and the hosts name no plugin: the composition root is the one place
// a plugin id is spelled out.
func TestNoCoreOrHostSourceSpellsOutAPluginID(t *testing.T) {
	roots := []string{
		"..",
		filepath.Join(numenModulesDir, "apps", "desktop"),
		filepath.Join(numenModulesDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for _, boxes := range getPluginIDLiterals(file) {
			t.Errorf("%s spells out a plugin id: %s", path, boxes)
		}
	})
}

// getAnonymousAssertions is every type assertion and type switch case of a file
// naming an interface that is spelled out where it is asserted.
func getAnonymousAssertions(file *ast.File) []token.Pos {
	var found []token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeAssertExpr:
			if _, anonymous := n.Type.(*ast.InterfaceType); anonymous {
				found = append(found, n.Pos())
			}
		case *ast.CaseClause:
			for _, one := range n.List {
				if _, anonymous := one.(*ast.InterfaceType); anonymous {
					found = append(found, one.Pos())
				}
			}
		}
		return true
	})
	return found
}

// What getAnonymousAssertions says is the whole of the rule, so it is asked
// directly.
func TestWhatTheAnonymousAssertionRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		src       string
		isRefused bool
	}{
		{`package p; func f(x any) { _ = x.(interface{ Stop() }) }`, true},
		{`package p; func f(x any) { _, _ = x.(interface{ Stop() }) }`, true},
		{`package p; func f(x any) { _ = x.(fmt.Stringer) }`, false},
		{`package p; func f(x any) { _ = x.(*Host) }`, false},
		{`package p; func f(x any) { switch x.(type) { case int: } }`, false},
		{`package p; func f(x any) { switch x.(type) { case interface{ Stop() }: } }`, true},
	} {
		if got := len(getAnonymousAssertions(parseSource(t, one.src))) > 0; got != one.isRefused {
			t.Errorf("%s: wantErr is %v, want %v", one.src, got, one.isRefused)
		}
	}
}

// The plugin manager is reached through the methods of port.PluginManager. The
// container and the hosts ask it for nothing it does not declare, so a value of
// it is never asserted to an interface spelled out at the assertion.
func TestNoHostOrContainerSourceAssertsToAnAnonymousInterface(t *testing.T) {
	roots := []string{
		".",
		filepath.Join(numenModulesDir, "apps", "desktop"),
		filepath.Join(numenModulesDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for range getAnonymousAssertions(file) {
			t.Errorf("%s asserts to an anonymous interface", path)
		}
	})
}

// getManifestIssues is every rule the manifest in data breaks, by the one
// validator the host and the plugin command line share.
func getManifestIssues(data []byte) []error {
	var manifest port.PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return []error{err}
	}
	return manifest.ListIssues()
}

// What getManifestIssues says is the whole of the rule, so it is asked
// directly.
func TestWhatTheManifestValidatorRefuses(t *testing.T) {
	valid := `{"id":"org.example.sample","name":"Sample","version":"1.0.0"}`
	if issues := getManifestIssues([]byte(valid)); len(issues) != 0 {
		t.Fatalf("a valid manifest has issues %v", issues)
	}
	for name, data := range map[string]string{
		"a single-segment id":         `{"id":"sample","name":"Sample","version":"1.0.0"}`,
		"a bad version":               `{"id":"org.example.sample","name":"Sample","version":"1.0"}`,
		"a document that is not JSON": `{`,
	} {
		if len(getManifestIssues([]byte(data))) == 0 {
			t.Errorf("%s is not rejected", name)
		}
	}
}

// Every manifest.json of a plugin passes the validator.
func TestEveryPluginManifestPassesTheValidator(t *testing.T) {
	found, err := filepath.Glob(filepath.Join(numenModulesDir, "plugins", "*", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(found) < 2 {
		t.Fatalf("%d manifests found: a rule checked against nothing", len(found))
	}
	for _, path := range found {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, issue := range getManifestIssues(data) {
			t.Errorf("%s: %v", path, issue)
		}
	}
}
