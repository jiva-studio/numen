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

func getDepguardGaps(config []byte) ([]string, error) {
	var parsed struct {
		Linters struct {
			Settings struct {
				Depguard struct {
					Rules map[string]load.DepguardRule `yaml:"rules"`
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
		load.NumenRoot + "libs",
		load.NumenRoot + "apps",
		"connectrpc.com/connect",
		"google.golang.org/protobuf",
	} {
		if !slices.Contains(denied, pkg) {
			gaps = append(gaps, "does not deny "+pkg)
		}
	}
	for _, pkg := range allowed {
		if strings.HasPrefix(pkg, load.NumenRoot) && !load.IsSDK(pkg) {
			gaps = append(gaps, "allows "+pkg)
		}
	}
	return gaps, nil
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

func getManifestIssues(data []byte) []error {
	parsed, err := manifestjson.Decode(data)
	if err != nil {
		return []error{err}
	}
	return parsed.ListIssues()
}
