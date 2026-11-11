package pluginindependence

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
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
	"gopkg.in/yaml.v3"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getPluginDependencyRefusal defines the whole rule, so it is tested directly.
func TestPluginDependencyRule_RejectsForeignDependencies(t *testing.T) {
	self := load.PluginsPrefix + "sample"
	for _, one := range []struct {
		dep       string
		isRefused bool
	}{
		{load.PluginsPrefix + "other", true},
		{load.PluginsPrefix + "other/backend/service", true},
		{load.Module + "port", true},
		{load.Module + "domain/plugin", true},
		{strings.TrimSuffix(load.Module, "/"), true},
		{load.NumenRoot + "apps/desktop/app", true},
		{self, false},
		{self + "/backend/service", false},
		{load.SdkModule, false},
		{load.SdkModule + "/vault", false},
		{load.SdkModule + "/server", false},
		{load.Wire, true},
		{load.Wire + "/gen/numen/v1/numenv1connect", true},
		{"connectrpc.com/connect", true},
		{"google.golang.org/protobuf/proto", true},
		{"google.golang.org/protobufx", false},
		{load.NotesModule, true},
		{load.NotesModule + "/domain", true},
		{load.NotesModule + "/edit", true},
		{load.NumenRoot + "libs/ui/workbench", true},
		{load.NumenRoot + "tools/lint-layers", true},
		{"golang.org/x/net/html", false},
		{"github.com/jiva-studio/other", false},
	} {
		if got := load.GetPluginDependencyRefusal(self, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s names %s: failure is %v, want %v", self, one.dep, got, one.isRefused)
		}
	}
}

// A plugin imports the plugin SDK and third-party code, in its sources and in its tests, under
// every build constraint.
func TestNoPluginImportsAnotherPluginOrTheCore(t *testing.T) {
	for _, dir := range load.ListPluginModules(t) {
		mod := load.PluginsPrefix + filepath.Base(dir)
		imports, err := load.GetImports(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(imports) == 0 {
			t.Fatalf("%s imports nothing: a rule checked against nothing", dir)
		}
		for path, deps := range imports {
			for _, dep := range deps {
				if why := load.GetPluginDependencyRefusal(mod, dep); why != "" {
					t.Errorf("%s imports %s: %s", path, dep, why)
				}
			}
		}
	}
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
          allow: ["` + load.SdkModule + `"]
          deny:
            - pkg: "` + load.NumenRoot + `libs"
            - pkg: "` + load.NumenRoot + `apps"
            - pkg: "connectrpc.com/connect"
            - pkg: "google.golang.org/protobuf"
`
	gaps, err := getDepguardGaps([]byte(complete))
	if err != nil || len(gaps) != 0 {
		t.Fatalf("a complete configuration has gaps %v (%v)", gaps, err)
	}
	missingRule := strings.Replace(complete, "$all", "**/other/**", 1)
	missingDeny := strings.Replace(complete, "connectrpc.com/connect", "example.com/other", 1)
	extraAllow := strings.Replace(complete, `allow: ["`+load.SdkModule+`"]`, `allow: ["`+load.NotesModule+`"]`, 1)
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
	dirs := append([]string{filepath.Join(load.RepoRootDir, "plugins", ".example")}, load.ListPluginModules(t)...)
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

// The core and the hosts name no plugin: the composition root is the one place
// a plugin id is spelled out.
func TestNoCoreOrHostSourceSpellsOutAPluginID(t *testing.T) {
	roots := []string{
		"..",
		filepath.Join(load.RepoRootDir, "apps", "desktop"),
		filepath.Join(load.RepoRootDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for _, boxes := range getPluginIDLiterals(file) {
			t.Errorf("%s spells out a plugin id: %s", path, boxes)
		}
	})
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

// The plugin host is reached through the capabilities of container.PluginHost. The
// container and the hosts ask it for nothing they do not declare, so a value of
// it is never asserted to an interface spelled out at the assertion.
func TestNoHostOrContainerSourceAssertsToAnAnonymousInterface(t *testing.T) {
	roots := []string{
		".",
		filepath.Join(load.RepoRootDir, "apps", "desktop"),
		filepath.Join(load.RepoRootDir, "apps", "mobile"),
	}
	parseSources(t, roots, func(path string, file *ast.File) {
		for range getAnonymousAssertions(file) {
			t.Errorf("%s asserts to an anonymous interface", path)
		}
	})
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
	found, err := filepath.Glob(filepath.Join(load.RepoRootDir, "plugins", "*", "manifest.json"))
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
	sharedPath := filepath.Join(load.RepoRootDir, "plugins", "shared")
	if _, err := os.Stat(sharedPath); !os.IsNotExist(err) {
		t.Fatalf("%s exists: plugins must be autonomous and cannot share internal libraries", sharedPath)
	}
}
