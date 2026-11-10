package container

import (
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

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

func checkExhaustructIncludesMatchTypes(t *testing.T, workspace workspacePackages) {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(repoRootDir, ".golangci-structure.yml"))
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
		names = append(names, getTypeNames(listProductionPackages(pkgs))...)
	}
	for _, pattern := range getUnmatchedPatterns(patterns, names) {
		t.Errorf("the exhaustruct include pattern %s matches no type of the workspace", pattern)
	}
}

// A pattern that matches no type is reported, and one that matches a type is not.
func TestExhaustructIncludeRule_ReportsAPatternThatMatchesNoType(t *testing.T) {
	names := []string{"example.com/a/domain/model.Model", "example.com/a/domain/model.Vector"}
	got := getUnmatchedPatterns([]string{`example\.com/a/domain/model\.Model$`, `example\.com/a/core/port\.Model$`}, names)
	if len(got) != 1 || got[0] != `example\.com/a/core/port\.Model$` {
		t.Errorf("unmatched patterns %v, want only the pattern of the missing package", got)
	}
}
