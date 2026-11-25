// Package pluginindependence is the check `go/plugin-independence`, run on the tree with the planted violations that prove it.
package pluginindependence

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{load.NumenRoot + "libs/shared/atomicfile", false},
		{load.NumenRoot + "libs/shared/onnx", false},
		{load.NumenRoot + "libs/shared/tokenizer", false},
		{load.NumenRoot + "libs/shared/tokenizer/x", false},
		{load.NumenRoot + "libs/sharedx", true},
		{load.NumenRoot + "libs/sharedx/onnx", true},
		{load.NumenRoot + "libs/plugin/catalog", true},
		{load.NumenRoot + "libs/shared/protocol/gen/numen/v1", true},
		{load.SyncServerModule + "/fakeserver", false},
		{"golang.org/x/net/html", false},
		{"github.com/jiva-studio/other", false},
	} {
		if got := load.GetPluginDependencyRefusal(self, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s names %s: failure is %v, want %v", self, one.dep, got, one.isRefused)
		}
	}
}

// A plugin imports the plugin SDK, libs/shared and third-party code, in its sources and in its tests,
// under every build constraint.
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
            - pkg: "` + load.NotesModule + `"
            - pkg: "` + load.Wire + `"
            - pkg: "` + load.NumenRoot + `apps"
            - pkg: "connectrpc.com/connect"
            - pkg: "google.golang.org/protobuf"
`
	gaps, err := getDepguardGaps([]byte(complete))
	if err != nil || len(gaps) != 0 {
		t.Fatalf("a complete configuration has gaps %v (%v)", gaps, err)
	}
	shared := strings.Replace(complete, `allow: ["`+load.SdkModule+`"]`, `allow: ["`+load.SdkModule+`", "`+load.NumenRoot+`libs/shared/"]`, 1)
	if gaps, err := getDepguardGaps([]byte(shared)); err != nil || len(gaps) != 0 {
		t.Errorf("a configuration that allows the shared libraries has gaps %v (%v)", gaps, err)
	}
	missingRule := strings.Replace(complete, "$all", "**/other/**", 1)
	missingDeny := strings.Replace(complete, "connectrpc.com/connect", "example.com/other", 1)
	missingNotesDeny := strings.Replace(complete, "            - pkg: \""+load.NotesModule+"\"\n", "", 1)
	missingWireDeny := strings.Replace(complete, "            - pkg: \""+load.Wire+"\"\n", "", 1)
	extraAllow := strings.Replace(complete, `allow: ["`+load.SdkModule+`"]`, `allow: ["`+load.NotesModule+`"]`, 1)
	extraPrefix := strings.Replace(complete, `allow: ["`+load.SdkModule+`"]`, `allow: ["`+load.SdkModule+`", "`+load.NumenRoot+`libs/sharedx"]`, 1)
	// depguard matches an allow by prefix, so libs/shared without the slash would also allow libs/sharedx.
	slashless := strings.Replace(complete, `allow: ["`+load.SdkModule+`"]`, `allow: ["`+load.SdkModule+`", "`+load.NumenRoot+`libs/shared"]`, 1)
	for name, config := range map[string]string{
		"a configuration without a rule over $all":                missingRule,
		"a rule that leaves a boundary out":                       missingDeny,
		"a rule that lets a plugin name the notes library":        missingNotesDeny,
		"a rule that lets a plugin name the protocol":             missingWireDeny,
		"a rule that allows another library":                      extraAllow,
		"a rule that allows a library named like the shared ones": extraPrefix,
		"a rule that allows libs/shared without the slash":        slashless,
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

// The container and the application hosts use the interfaces declared by their consumers.
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

// TestManifestList_FindsThePlantedManifests verifies that the list finds the manifest of each
// plugin folder and nothing else, and fails on a tree without one.
func TestManifestList_FindsThePlantedManifests(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"plugins/alpha/manifest.json": "{}",
		"plugins/beta/manifest.json":  "{}",
		"plugins/beta/other.json":     "{}",
		"docs/manifest.json":          "{}",
	})
	found, err := listManifests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Errorf("found %v, want the manifests of alpha and beta", found)
	}
	if _, err := listManifests(t.TempDir()); err == nil {
		t.Error("a tree without a manifest passed")
	}
}

func listManifests(root string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(root, "plugins", "*", "manifest.json"))
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no plugin manifest under %s/plugins: a rule checked against nothing", root)
	}
	return found, nil
}

func TestEveryPluginManifestPassesTheValidator(t *testing.T) {
	found, err := listManifests(load.RepoRootDir)
	if err != nil {
		t.Fatal(err)
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

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
