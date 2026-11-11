package pluginadapterlayers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"gopkg.in/yaml.v3"
)

const pluginRootImportPath = "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"

func getPluginRootImports(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			if to, err := strconv.Unquote(imp.Path.Value); err == nil && to == pluginRootImportPath {
				found = append(found, path)
			}
		}
		return nil
	})
	return found, err
}

func hasDenied(rule load.DepguardRule, pkg string) bool {
	for _, deny := range rule.Deny {
		if deny.Pkg == pkg {
			return true
		}
	}
	return false
}

func getDepguardRules(t *testing.T) map[string]load.DepguardRule {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Linters struct {
			Settings struct {
				Depguard struct {
					Rules map[string]load.DepguardRule `yaml:"rules"`
				} `yaml:"depguard"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	return config.Linters.Settings.Depguard.Rules
}

func getDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}
