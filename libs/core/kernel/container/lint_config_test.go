package container

import (
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

func listLintConfigs(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); name == "node_modules" || name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".golangci") && strings.HasSuffix(entry.Name(), ".yml") {
			found = append(found, path)
		}
		return nil
	})
	slices.Sort(found)
	return found, err
}

func readExhaustructPatterns(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config struct {
		Linters struct {
			Settings struct {
				Exhaustruct struct {
					Include []string `yaml:"include"`
					Exclude []string `yaml:"exclude"`
				} `yaml:"exhaustruct"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	exhaustruct := config.Linters.Settings.Exhaustruct
	return append(slices.Clone(exhaustruct.Include), exhaustruct.Exclude...), nil
}

func listStructNames(pkgs []*packages.Package) []string {
	var names []string
	for _, pkg := range listProductionPackages(pkgs) {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || object.IsAlias() {
				continue
			}
			if _, ok := object.Type().Underlying().(*types.Struct); ok {
				names = append(names, pkg.PkgPath+"."+name)
			}
		}
	}
	return names
}

func getDeadPatterns(patterns, names []string) ([]string, error) {
	var dead []string
	for _, pattern := range patterns {
		expr, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		if !slices.ContainsFunc(names, expr.MatchString) {
			dead = append(dead, pattern)
		}
	}
	return dead, nil
}

func checkEveryExhaustructPatternMatchesAType(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var names []string
	for _, pkgs := range workspace {
		names = append(names, listStructNames(pkgs)...)
	}
	root, err := filepath.Abs(repoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	configs, err := listLintConfigs(root)
	if err != nil {
		t.Fatal(err)
	}
	patterns := 0
	for _, config := range configs {
		list, err := readExhaustructPatterns(config)
		if err != nil {
			t.Fatal(err)
		}
		patterns += len(list)
		dead, err := getDeadPatterns(list, names)
		if err != nil {
			t.Fatalf("%s: %v", config, err)
		}
		for _, one := range dead {
			t.Errorf("%s: the exhaustruct pattern %s matches no struct type of the repository", config, one)
		}
	}

	// A walk that reads no configuration or no pattern checks nothing.
	if len(configs) < 20 || patterns < 20 || len(names) < 1000 {
		t.Fatalf("%d configurations, %d patterns and %d struct types read: the rule is checked against nothing", len(configs), patterns, len(names))
	}
}

func TestExhaustructPatternRule_RejectsAPatternThatMatchesNoType(t *testing.T) {
	dead, err := getDeadPatterns(
		[]string{`.*/core/port/\w+port\.(\w+Model|Vector)$`, `.*/kernel/domain/embedding\.(Model|Vector)$`, `fixture/.*`},
		[]string{"github.com/x/libs/core/kernel/domain/embedding.Model", "fixture/a.T"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{`.*/core/port/\w+port\.(\w+Model|Vector)$`}; !slices.Equal(dead, want) {
		t.Errorf("the rule rejects %v, want %v", dead, want)
	}
}

var mutationDirective = regexp.MustCompile(`^//\s*gomutants:disable(-next-line)?(\s+[A-Z_]+(,[A-Z_]+)*)?\s+reason="((?:\\.|[^"\\])*)"`)

func getMutationDirectives(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); name == "node_modules" || name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		lines, readErr := getDirectiveLines(path)
		for _, one := range lines {
			found = append(found, filepath.ToSlash(rel)+":"+one)
		}
		return readErr
	})
	return found, err
}

func getUnexplainedDirectives(directives []string) []string {
	var found []string
	for _, one := range directives {
		_, text, _ := strings.Cut(one, ": ")
		match := mutationDirective.FindStringSubmatch(text)
		if match == nil || len(strings.Fields(match[4])) < 3 {
			found = append(found, one)
		}
	}
	return found
}

func TestMutationDirectiveRule_RejectsMissingReasons(t *testing.T) {
	root, err := filepath.Abs(repoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	directives, err := getMutationDirectives(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range getUnexplainedDirectives(directives) {
		t.Errorf("%s: a gomutants directive gives a reason of three words or more", one)
	}
	if len(directives) == 0 {
		t.Fatalf("%d directives read: the walk is not reading the repository", len(directives))
	}
}

func TestMutationDirectiveRule_RejectsADirectiveWithoutAReason(t *testing.T) {
	got := getUnexplainedDirectives([]string{
		`a.go:1: // gomutants:disable-next-line`,
		`a.go:2: // gomutants:disable-next-line BRANCH_IF`,
		`a.go:3: // gomutants:disable-next-line BRANCH_IF reason=""`,
		`a.go:4: // gomutants:disable-next-line BRANCH_IF reason="fine"`,
		`a.go:5: // gomutants:disable-next-line BRANCH_IF reason="the constant page always parses"`,
	})
	want := []string{
		`a.go:1: // gomutants:disable-next-line`,
		`a.go:2: // gomutants:disable-next-line BRANCH_IF`,
		`a.go:3: // gomutants:disable-next-line BRANCH_IF reason=""`,
		`a.go:4: // gomutants:disable-next-line BRANCH_IF reason="fine"`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func getDirectiveLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var found []string
	for i, line := range strings.Split(string(raw), "\n") {
		text := strings.TrimSpace(line)
		if strings.HasPrefix(text, "//") && strings.Contains(text, "gomutants:disable") {
			found = append(found, strconv.Itoa(i+1)+": "+text)
		}
	}
	return found, nil
}
