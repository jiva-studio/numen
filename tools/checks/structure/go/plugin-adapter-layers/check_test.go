package pluginadapterlayers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"gopkg.in/yaml.v3"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestNoPluginSubpackageImportsThePluginRoot(t *testing.T) {
	root := "../internal/adapter/plugin"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		found, err := getPluginRootImports(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range found {
			t.Errorf("%s imports the plugin root package: the root assembles its subpackages, and they never reach back up to it", path)
		}
	}
}

func TestPluginRootImportIsFoundInASubpackage(t *testing.T) {
	dir := t.TempDir()
	source := "package sample\n\nimport _ \"" + pluginRootImportPath + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	found, err := getPluginRootImports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || filepath.Base(found[0]) != "sample.go" {
		t.Errorf("found %v, want only sample.go", found)
	}
}

// Every adapter beside the plugin adapter is denied to it, so a new adapter cannot be reached by plugin code
// until the rule is edited.
func TestPluginIsolationRuleDeniesEverySiblingAdapter(t *testing.T) {
	rule, found := getDepguardRules(t)["plugin-adapter-isolation"]
	if !found {
		t.Fatal("the plugin-adapter-isolation rule is missing from .golangci.yml")
	}
	for _, name := range getDirNames(t, "../adapter") {
		if name != "plugin" && !hasDenied(rule, load.CoreImportPath+"/adapter/"+name) {
			t.Errorf("adapter/%s is an adapter and the plugin-adapter-isolation rule does not deny it", name)
		}
	}
	for _, name := range getDirNames(t, "../internal/adapter") {
		if name != "plugin" && !hasDenied(rule, load.CoreImportPath+"/internal/adapter/"+name) {
			t.Errorf("internal/adapter/%s is an adapter and the plugin-adapter-isolation rule does not deny it", name)
		}
	}
}

// Every bridge and every host package is on the rule of its side, and each side denies every package of the other.
func TestPluginSideRulesNameEveryPackage(t *testing.T) {
	rules := getDepguardRules(t)
	hosts := getDirNames(t, "../adapter/plugin")
	var bridges []string
	for _, name := range getDirNames(t, "../internal/adapter/plugin") {
		if strings.HasSuffix(name, "bridge") && name != "bridge" {
			bridges = append(bridges, name)
		}
	}
	if len(bridges) == 0 || len(hosts) == 0 {
		t.Fatalf("found bridges %v and hosts %v: the plugin adapter is not being read", bridges, hosts)
	}
	denied := func(rule load.DepguardRule, rest []string, ruleName string) {
		for _, name := range rest {
			importPath := pluginRootImportPath + "/" + name
			if slices.Contains(hosts, name) {
				importPath = load.CoreImportPath + "/adapter/plugin/" + name
			}
			if !hasDenied(rule, importPath) {
				t.Errorf("rule %s does not deny %s", ruleName, name)
			}
		}
	}
	for _, name := range bridges {
		ruleName := "plugin-" + name
		rule, found := rules[ruleName]
		if !found {
			t.Errorf("%s is a bridge and .golangci.yml has no rule %s", name, ruleName)
			continue
		}
		if !slices.Contains(rule.Files, "**/internal/adapter/plugin/"+name+"/**") {
			t.Errorf("rule %s does not list the files of %s", ruleName, name)
		}
		others := slices.DeleteFunc(slices.Clone(bridges), func(other string) bool { return other == name })
		denied(rule, append(others, hosts...), ruleName)
	}
	hostRule, found := rules["plugin-host-side"]
	if !found {
		t.Fatal("the plugin-host-side rule is missing from .golangci.yml")
	}
	for _, name := range hosts {
		if !slices.Contains(hostRule.Files, "**/adapter/plugin/"+name+"/**") {
			t.Errorf("rule plugin-host-side does not list the files of %s", name)
		}
	}
	denied(hostRule, append([]string{"bridge"}, bridges...), "plugin-host-side")
}
