package domainlayer

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The packages under domain/ of the core import the standard library, the notes domain, the manifest of the plugin catalog and each other.
// The walk reads every non-test file under domain/ and fails when it reads fewer than the domain holds.
func TestNoDomainPackageImportsThirdPartyCodeOrAnotherLayer(t *testing.T) {
	root := filepath.Join("..", "domain")
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
				t.Errorf("%s imports %s: %s", path, to, why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if read < 30 {
		t.Fatalf("%d files under domain/ read: the rule is checked against nothing", read)
	}
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
	found := 0
	for _, root := range []string{
		filepath.Join("..", "..", "..", "plugin", "sdk-go"),
		filepath.Join("..", "..", "..", "plugin", "catalog"),
	} {
		imports, err := load.GetImports(root)
		if err != nil {
			t.Fatal(err)
		}
		found += len(imports)
		for path, deps := range imports {
			for _, dep := range deps {
				if why := getCoreImportRefusal(dep); why != "" {
					t.Errorf("%s imports %s: %s", path, dep, why)
				}
			}
		}
	}
	if found < 50 {
		t.Fatalf("%d files of the SDK and the catalog read: the walk is not reading them", found)
	}
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
