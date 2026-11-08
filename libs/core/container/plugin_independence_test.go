package container

import (
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

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"

	"gopkg.in/yaml.v3"
)

// numenRoot is what every module of this repository is named under.
const numenRoot = "github.com/jiva-studio/numen/"

// notesModule is the module of the note model, which plugins do not name.
const notesModule = numenRoot + "libs/notes"

// repoRootDir is the repository root, relative to this package.
const repoRootDir = "../../.."

func isWithin(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

func getPluginDependencyRefusal(mod, dep string) string {
	switch {
	case isWithin(dep, mod), isSDK(dep):
		return ""
	case isWithin(dep, "connectrpc.com/connect"), isWithin(dep, "google.golang.org/protobuf"):
		return "a plugin calls and answers the core through the plugin SDK, not through the RPC library"
	case !strings.HasPrefix(dep, numenRoot):
		return ""
	case isWithin(dep, notesModule):
		return "a plugin names no package of the notes library; it reaches notes through the plugin SDK"
	case isPlugin(dep):
		return "a plugin names no other plugin"
	case isWithin(dep, strings.TrimSuffix(module, "/")):
		return "a plugin names no package of the core"
	}
	return "a plugin names only the plugin SDK and third-party code"
}

// getPluginDependencyRefusal defines the whole rule, so it is tested directly.
func TestPluginDependencyRule_RejectsForeignDependencies(t *testing.T) {
	self := pluginsPrefix + "sample"
	for _, one := range []struct {
		dep       string
		isRefused bool
	}{
		{pluginsPrefix + "other", true},
		{pluginsPrefix + "other/backend/service", true},
		{module + "port", true},
		{strings.TrimSuffix(module, "/"), true},
		{numenRoot + "apps/desktop/app", true},
		{self, false},
		{self + "/backend/service", false},
		{sdkModule, false},
		{sdkModule + "/vault", false},
		{sdkModule + "/server", false},
		{wire, true},
		{wire + "/gen/numen/v1/numenv1connect", true},
		{"connectrpc.com/connect", true},
		{"google.golang.org/protobuf/proto", true},
		{"google.golang.org/protobufx", false},
		{notesModule, true},
		{notesModule + "/domain", true},
		{notesModule + "/edit", true},
		{numenRoot + "libs/ui/shell", true},
		{numenRoot + "tools/lint-layers", true},
		{"golang.org/x/net/html", false},
		{"github.com/jiva-studio/other", false},
	} {
		if got := getPluginDependencyRefusal(self, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s names %s: failure is %v, want %v", self, one.dep, got, one.isRefused)
		}
	}
}

func listPluginModules(t *testing.T) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(repoRootDir, "plugins", "*", "go.mod"))
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

// A plugin imports the plugin SDK and third-party code, in its sources and in its tests, under
// every build constraint.
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
	Allow []string `yaml:"allow"`
	Deny  []struct {
		Pkg string `yaml:"pkg"`
	} `yaml:"deny"`
}

func getDepguardGaps(config []byte) ([]string, error) {
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
	var denied, allowed []string
	for _, rule := range parsed.Linters.Settings.Depguard.Rules {
		if !slices.Contains(rule.Files, "$all") {
			continue
		}
		for _, deny := range rule.Deny {
			denied = append(denied, deny.Pkg)
		}
		allowed = append(allowed, rule.Allow...)
	}
	if denied == nil {
		return []string{"no depguard rule over $all"}, nil
	}
	var gaps []string
	for _, pkg := range []string{
		numenRoot + "libs",
		numenRoot + "apps",
		"connectrpc.com/connect",
		"google.golang.org/protobuf",
	} {
		if !slices.Contains(denied, pkg) {
			gaps = append(gaps, "does not deny "+pkg)
		}
	}
	for _, pkg := range allowed {
		if strings.HasPrefix(pkg, numenRoot) && !isSDK(pkg) {
			gaps = append(gaps, "allows "+pkg)
		}
	}
	return gaps, nil
}

// getDepguardGaps defines the whole rule, so it is tested directly.
func TestWhatTheDepguardRulesRefuse(t *testing.T) {
	complete := `
linters:
  settings:
    depguard:
      rules:
        plugin-isolation:
          files: ["$all"]
          allow: ["` + sdkModule + `"]
          deny:
            - pkg: "` + numenRoot + `libs"
            - pkg: "` + numenRoot + `apps"
            - pkg: "connectrpc.com/connect"
            - pkg: "google.golang.org/protobuf"
`
	gaps, err := getDepguardGaps([]byte(complete))
	if err != nil || len(gaps) != 0 {
		t.Fatalf("a complete configuration has gaps %v (%v)", gaps, err)
	}
	missingRule := strings.Replace(complete, "$all", "**/other/**", 1)
	missingDeny := strings.Replace(complete, "connectrpc.com/connect", "example.com/other", 1)
	extraAllow := strings.Replace(complete, `allow: ["`+sdkModule+`"]`, `allow: ["`+notesModule+`"]`, 1)
	for name, config := range map[string]string{
		"a configuration without a rule over $all": missingRule,
		"a rule that leaves a boundary out":        missingDeny,
		"a rule that allows another library":       extraAllow,
	} {
		if gaps, err := getDepguardGaps([]byte(config)); err != nil || len(gaps) == 0 {
			t.Errorf("%s is not found (%v)", name, err)
		}
	}
}

// Every plugin module and the golden template have a depguard rule that denies
// every library but the plugin SDK, the applications and the RPC library.
func TestPluginModules_HaveDepguardRulesDenyingCoreAndPlugins(t *testing.T) {
	dirs := append([]string{filepath.Join(repoRootDir, "plugins", ".example")}, listPluginModules(t)...)
	for _, dir := range dirs {
		path := filepath.Join(dir, ".golangci.yml")
		config, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		gaps, err := getDepguardGaps(config)
		if err != nil {
			t.Fatal(err)
		}
		for _, gap := range gaps {
			t.Errorf("%s: %s", path, gap)
		}
	}
}

// pluginIDLiteral is a plugin id spelled out: the namespace of the plugins that
// ship with numen, and a name after it.
var pluginIDLiteral = regexp.MustCompile(`md\.numen\.[A-Za-z0-9]`)

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

// getPluginIDLiterals defines the whole rule, so it is tested directly.
func TestPluginIDRule_RejectsPluginIDLiterals(t *testing.T) {
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

func listSources(root string) ([]string, error) {
	var sources []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "plugintest" {
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
		filepath.Join(repoRootDir, "apps", "desktop"),
		filepath.Join(repoRootDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for _, boxes := range getPluginIDLiterals(file) {
			t.Errorf("%s spells out a plugin id: %s", path, boxes)
		}
	})
}

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

// getAnonymousAssertions defines the whole rule, so it is tested directly.
func TestAnonymousAssertionRule_RejectsInterfaceAssertions(t *testing.T) {
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
		filepath.Join(repoRootDir, "apps", "desktop"),
		filepath.Join(repoRootDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for range getAnonymousAssertions(file) {
			t.Errorf("%s asserts to an anonymous interface", path)
		}
	})
}

func getManifestIssues(data []byte) []error {
	parsed, err := manifestjson.Decode(data)
	if err != nil {
		return []error{err}
	}
	return parsed.ListIssues()
}

// getManifestIssues defines the whole rule, so it is tested directly.
func TestManifestValidator_RejectsInvalidManifests(t *testing.T) {
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
	found, err := filepath.Glob(filepath.Join(repoRootDir, "plugins", "*", "manifest.json"))
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

// Plugins must be autonomous; plugins/ contains no shared/ folder.
func TestNoSharedPluginDirectoryExists(t *testing.T) {
	sharedPath := filepath.Join(repoRootDir, "plugins", "shared")
	if _, err := os.Stat(sharedPath); !os.IsNotExist(err) {
		t.Fatalf("%s exists: plugins must be autonomous and cannot share internal libraries", sharedPath)
	}
}
