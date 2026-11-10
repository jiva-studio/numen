package container

import (
	"bufio"
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

// listLintConfigs returns every golangci-lint configuration of the repository.
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

// listStructNames returns "import path.Name" for every struct type the packages declare, which is how
// exhaustruct matches its include and exclude patterns.
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

// getDeadPatterns returns the patterns that match none of the names.
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

// maxMutationDirectives is how many comments switch mutation testing off in the whole repository. Each
// one accepts a mutant that no test kills, so the number only falls: a new one needs an old one removed
// by a test that kills its mutant.
const maxMutationDirectives = 850

var mutationDirective = regexp.MustCompile(`^//\s*gomutants:disable(-next-line)?\s+[A-Z_]+(,[A-Z_]+)*\s+reason="([^"]*)"`)

// getMutationDirectives returns the comment lines of Go source files that switch mutation testing off,
// each as "file:line: text".
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
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for line := 1; scanner.Scan(); line++ {
			text := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(text, "//") && strings.Contains(text, "gomutants:disable") {
				rel, _ := filepath.Rel(root, path)
				found = append(found, filepath.ToSlash(rel)+":"+strconv.Itoa(line)+": "+text)
			}
		}
		return scanner.Err()
	})
	return found, err
}

// getUnexplainedDirectives returns the directives that name no mutator or give no reason of at least
// three words.
func getUnexplainedDirectives(directives []string) []string {
	var found []string
	for _, one := range directives {
		_, text, _ := strings.Cut(one, ": ")
		match := mutationDirective.FindStringSubmatch(text)
		if match == nil || len(strings.Fields(match[3])) < 3 {
			found = append(found, one)
		}
	}
	return found
}

func TestMutationDirectivesGiveAReasonAndStayUnderTheCap(t *testing.T) {
	root, err := filepath.Abs(repoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	directives, err := getMutationDirectives(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range getUnexplainedDirectives(directives) {
		t.Errorf("%s: a gomutants directive names its mutator and gives a reason of three words or more", one)
	}
	if len(directives) > maxMutationDirectives {
		t.Errorf("%d gomutants directives, more than the %d allowed", len(directives), maxMutationDirectives)
	}
	if len(directives) < 100 {
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
