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

// notesDomain is the package of the note model that a domain package of the core may import.
const notesDomain = notesModule + "/domain"

// catalogManifest is the library of the plugin contract that a domain package of the core may import; it
// never imports the core.
const catalogManifest = numenRoot + "libs/plugin/catalog/manifest"

func isStandardLibrary(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// getDomainImportRefusal returns why a package under domain/ may not import the package at an import
// path. Imports of the core itself are judged by getRefusal.
func getDomainImportRefusal(to string) string {
	switch {
	case isStandardLibrary(to), to == notesDomain, to == catalogManifest, strings.HasPrefix(to, module+"domain/"):
		return ""
	case strings.HasPrefix(to, module):
		return "a domain package names no layer of the core other than the domain"
	}
	return "a domain package imports the standard library, the notes domain, the manifest of the plugin catalog and the domain packages, and no third-party code"
}

// The packages under domain/ of the core import the standard library, the notes domain, the manifest of the plugin catalog and each other.
// The walk reads every non-test file under domain/; the folder does not exist before the first domain
// package is created, and then no file is read.
func TestNoDomainPackageImportsThirdPartyCodeOrAnotherLayer(t *testing.T) {
	root := filepath.Join("..", "domain")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("the core has no domain package yet")
	}
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
	if read == 0 {
		t.Fatal("no file under domain/ read: the rule is checked against nothing")
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
		{module + "domain/settings", false},
		{module + "port/pluginport", true},
		{module + "usecase/note", true},
		{module + "adapter/index", true},
		{module + "internal/adapter/trash", true},
		{module + "container", true},
		{notesModule + "/edit", true},
		{numenRoot + "libs/plugin/catalog/when", true},
		{catalogManifest + "/internal", true},
		{sdkModule + "/vault", true},
		{"github.com/google/uuid", true},
		{"golang.org/x/text/unicode/norm", true},
		{"gopkg.in/yaml.v3", true},
	} {
		if got := getDomainImportRefusal(tc.to) != ""; got != tc.isRefused {
			t.Errorf("a domain package imports %s: refused %v, want %v", tc.to, got, tc.isRefused)
		}
	}
}

// getCoreImportRefusal returns why the plugin SDK or the plugin catalog may not import the package at an
// import path.
func getCoreImportRefusal(to string) string {
	if strings.HasPrefix(to, module) || to == strings.TrimSuffix(module, "/") {
		return "the plugin SDK and the plugin catalog import no package of the core, a domain package included"
	}
	return ""
}

// No Go file of the plugin SDK or of the plugin catalog imports a package of the core.
func TestNoSDKOrCatalogPackageImportsTheCore(t *testing.T) {
	found := 0
	for _, root := range []string{
		filepath.Join("..", "..", "..", "plugin", "sdk-go"),
		filepath.Join("..", "..", "..", "plugin", "catalog"),
	} {
		imports, err := getImports(root)
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
		{module + "domain/plugin", true},
		{module + "port/pluginport", true},
		{module + "usecase/note", true},
		{strings.TrimSuffix(module, "/"), true},
		{sdkModule + "/vault", false},
		{numenRoot + "libs/plugin/catalog/manifest", false},
		{wire, false},
		{"errors", false},
	} {
		if got := getCoreImportRefusal(tc.to) != ""; got != tc.isRefused {
			t.Errorf("the SDK imports %s: refused %v, want %v", tc.to, got, tc.isRefused)
		}
	}
}
