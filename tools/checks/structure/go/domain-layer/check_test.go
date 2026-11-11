// Package domainlayer is the check `go/domain-layer`, run on the tree with the planted violations that prove it.
package domainlayer

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The packages under domain/ of the core import the standard library, the notes domain, the manifest of the plugin catalog and each other.
// The walk reads every non-test file under domain/ and fails when it reads none.
func TestNoDomainPackageImportsThirdPartyCodeOrAnotherLayer(t *testing.T) {
	wrong, err := findDomainImportRefusals(filepath.Join("..", "domain"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// TestDomainWalk_FindsThePlantedImports verifies that the walk finds an import of an adapter and of
// third-party code in a domain file and passes the standard library.
func TestDomainWalk_FindsThePlantedImports(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"settings/settings.go": "package settings\n\nimport (\n\t\"errors\"\n\t\"github.com/google/uuid\"\n\t\"" + load.Module + "adapter/index\"\n)\n",
	})
	wrong, err := findDomainImportRefusals(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 2 {
		t.Errorf("the walk found %v, want the imports of uuid and of adapter/index", wrong)
	}
}

// TestDomainWalk_FailsOnAnEmptyTree verifies that a root without a Go file is an error, not a pass.
func TestDomainWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findDomainImportRefusals(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a Go file passed")
	}
}

func findDomainImportRefusals(root string) ([]string, error) {
	// Fails when it reads no file.
	var wrong []string
	var read int
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, spec := range file.Imports {
			to, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if why := getDomainImportRefusal(to); why != "" {
				wrong = append(wrong, fmt.Sprintf("%s imports %s: %s", path, to, why))
			}
		}
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no Go file read under %s: the walk checks nothing", root)
	}
	return wrong, err
}

// getDomainImportRefusal defines the rule over import paths, so it is tested directly.
func TestDomainImportRule_RejectsEverythingButTheStandardLibraryTheNotesDomainTheManifestAndTheDomain(t *testing.T) {
	for _, tc := range []struct {
		to        string
		isRefused bool
	}{
		{"errors", false},
		{"net/url", false},
		{"time", false},
		{notesDomain, false},
		{catalogManifest, false},
		{load.Module + "domain/settings", false},
		{load.Module + "port/pluginport", true},
		{load.Module + "usecase/note", true},
		{load.Module + "adapter/index", true},
		{load.Module + "internal/adapter/themefiles", true},
		{load.Module + "internal/adapter/trash", true},
		{load.Module + "container", true},
		{load.NotesModule + "/edit", true},
		{load.NumenRoot + "libs/plugin/catalog/when", true},
		{catalogManifest + "/internal", true},
		{load.SdkModule + "/vault", true},
		{"github.com/google/uuid", true},
		{"golang.org/x/text/unicode/norm", true},
		{"gopkg.in/yaml.v3", true},
	} {
		if got := getDomainImportRefusal(tc.to) != ""; got != tc.isRefused {
			t.Errorf("a domain package imports %s: refused %v, want %v", tc.to, got, tc.isRefused)
		}
	}
}

// No Go file of the plugin SDK or of the plugin catalog imports a package of the core.
func TestNoSDKOrCatalogPackageImportsTheCore(t *testing.T) {
	for _, root := range []string{
		filepath.Join("..", "..", "..", "plugin", "sdk-go"),
		filepath.Join("..", "..", "..", "plugin", "catalog"),
	} {
		wrong, err := findCoreImports(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range wrong {
			t.Error(one)
		}
	}
}

// TestCoreImportWalk_FindsThePlantedImport verifies that the walk finds an import of the core in a
// file of the SDK.
func TestCoreImportWalk_FindsThePlantedImport(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"vault/vault.go": "package vault\n\nimport _ \"" + load.Module + "port/pluginport\"\n",
		"agent/agent.go": "package agent\n\nimport _ \"errors\"\n",
	})
	wrong, err := findCoreImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "vault/vault.go imports") {
		t.Errorf("the walk found %v, want the import in vault/vault.go only", wrong)
	}
}

// TestCoreImportWalk_FailsOnAnEmptyTree verifies that a root without an import is an error, not a pass.
func TestCoreImportWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findCoreImports(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a Go file passed")
	}
}

func findCoreImports(root string) ([]string, error) {
	imports, err := load.GetImports(root)
	if err != nil {
		return nil, err
	}
	if len(imports) == 0 {
		return nil, fmt.Errorf("no Go file with an import read under %s: the walk checks nothing", root)
	}
	var wrong []string
	for path, deps := range imports {
		for _, dep := range deps {
			if why := getCoreImportRefusal(dep); why != "" {
				wrong = append(wrong, fmt.Sprintf("%s imports %s: %s", path, dep, why))
			}
		}
	}
	slices.Sort(wrong)
	return wrong, nil
}

// getCoreImportRefusal defines the rule over import paths, so it is tested directly.
func TestCoreImportRule_RejectsEveryPackageOfTheCore(t *testing.T) {
	for _, tc := range []struct {
		to        string
		isRefused bool
	}{
		{load.Module + "domain/plugin", true},
		{load.Module + "port/pluginport", true},
		{load.Module + "usecase/note", true},
		{strings.TrimSuffix(load.Module, "/"), true},
		{load.SdkModule + "/vault", false},
		{load.NumenRoot + "libs/plugin/catalog/manifest", false},
		{load.Wire, false},
		{"errors", false},
	} {
		if got := getCoreImportRefusal(tc.to) != ""; got != tc.isRefused {
			t.Errorf("the SDK imports %s: refused %v, want %v", tc.to, got, tc.isRefused)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
