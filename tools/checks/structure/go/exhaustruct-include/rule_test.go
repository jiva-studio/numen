package exhaustructinclude

import (
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

type structureConfig struct {
	Linters struct {
		Settings struct {
			Exhaustruct struct {
				Include []string `yaml:"include"`
			} `yaml:"exhaustruct"`
		} `yaml:"settings"`
	} `yaml:"linters"`
}

func getTypeNames(pkgs []*packages.Package) []string {
	var names []string
	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if object, ok := scope.Lookup(name).(*types.TypeName); ok && !object.IsAlias() {
				names = append(names, pkg.PkgPath+"."+name)
			}
		}
	}
	return names
}

func getUnmatchedPatterns(patterns, names []string) []string {
	var unmatched []string
	for _, pattern := range patterns {
		expression := regexp.MustCompile(pattern)
		if !slices.ContainsFunc(names, expression.MatchString) {
			unmatched = append(unmatched, pattern)
		}
	}
	return unmatched
}

func checkExhaustructIncludesMatchTypes(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(load.RepoRootDir, ".golangci-structure.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config structureConfig
	if err := yaml.Unmarshal(text, &config); err != nil {
		t.Fatal(err)
	}
	patterns := config.Linters.Settings.Exhaustruct.Include
	if len(patterns) < 3 {
		t.Fatalf("%d exhaustruct include patterns read: the rule is checked against nothing", len(patterns))
	}
	var names []string
	for _, pkgs := range workspace {
		names = append(names, getTypeNames(load.ListProductionPackages(pkgs))...)
	}
	for _, pattern := range getUnmatchedPatterns(patterns, names) {
		t.Errorf("the exhaustruct include pattern %s matches no type of the workspace", pattern)
	}
}
