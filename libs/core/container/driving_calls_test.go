package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// wiring are the files of a driving adapter that bind the ports the composition
// root hands it. Under strict Clean Architecture, zero files are exempted:
// every driving adapter speaks only to usecase/* and never holds or calls a port directly.
var wiring []string

// getPortFields are the names of the struct fields a file types port.X.
func getPortFields(file *ast.File) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		target, is := node.(*ast.StructType)
		if !is {
			return true
		}
		for _, one := range target.Fields.List {
			callType := one.Type
			if star, is := callType.(*ast.StarExpr); is {
				callType = star.X
			}
			pos, is := callType.(*ast.SelectorExpr)
			if !is {
				continue
			}
			if from, is := pos.X.(*ast.Ident); is && isPortName(from.Name) {
				for _, name := range one.Names {
					found = append(found, name.Name)
				}
			}
		}
		return true
	})
	return found
}

// getFieldCalls are the calls in a file made as x.Field.Method(...) where Field
// is one of these names, each written Field.Method.
func getFieldCalls(file *ast.File, fields []string) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, is := node.(*ast.CallExpr)
		if !is {
			return true
		}
		method, is := call.Fun.(*ast.SelectorExpr)
		if !is {
			return true
		}
		field, is := method.X.(*ast.SelectorExpr)
		if is && slices.Contains(fields, field.Sel.Name) {
			found = append(found, field.Sel.Name+"."+method.Sel.Name)
		}
		return true
	})
	return found
}

// isDrivingAdapterFile says whether a path, relative to the core, is a file of
// an adapter something outside comes in through.
func isDrivingAdapterFile(rel string) bool {
	return isAmong([]string{"adapter/cli", "adapter/window"}, filepath.ToSlash(filepath.Dir(rel)))
}

// A driving adapter asks a use case. The port it holds in a field is the
// composition root's to hand over, and a call made through that field is the
// adapter doing the use case's work.
func TestNoDrivingAdapterCallsAPortOutsideItsWiring(t *testing.T) {
	fields := map[string][]string{}
	files := map[string]*ast.File{}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel := getPackage("..", path) + "/" + entry.Name()
		if !isDrivingAdapterFile(rel) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		files[rel] = file
		dir := filepath.Dir(rel)
		fields[dir] = append(fields[dir], getPortFields(file)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A walk that read no driving adapter, or met no port field, is a rule
	// checked against nothing.
	if len(files) < 30 {
		t.Fatalf("%d files of the driving adapters read: the walk is not reading them", len(files))
	}
	var target int
	for _, one := range fields {
		target += len(one)
	}
	if target == 0 {
		t.Fatal("no driving adapter holds a port in a field: the walk is not reading them")
	}

	for rel, file := range files {
		if slices.Contains(wiring, rel) {
			continue
		}
		calls := getFieldCalls(file, fields[filepath.Dir(rel)])
		if len(calls) == 0 {
			continue
		}
		t.Errorf("%s calls %s: a driving adapter asks a use case, not a port", rel, calls[0])
	}
	for _, rel := range wiring {
		if files[rel] == nil {
			t.Errorf("wiring holds %s and it is not there", rel)
		}
	}
}

// call through a field typed port.X, and lets through a call through a field of
// another type and a call on a value that is no field.
func TestDrivingCallRule_RejectsPortFieldCalls(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "call.go", `package p

type Notes struct {
	Queries noteport.NoteQueries
	Links   *noteport.LinkQueries
	Find    usecase.Find
}

func read(a *API, queries noteport.NoteQueries) {
	a.Notes.Queries.GetTypes()
	a.Notes.Links.Resolve()
	a.Notes.Find.Execute()
	queries.GetTypes()
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	fields := getPortFields(file)
	if want := []string{"Queries", "Links"}; !slices.Equal(fields, want) {
		t.Fatalf("the port fields are %v, want %v", fields, want)
	}
	want := []string{"Queries.GetTypes", "Links.Resolve"}
	if got := getFieldCalls(file, fields); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// depguardFiles are the file patterns of the lint rule that keeps a driving
// adapter off the ports.
func depguardFiles(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Linters struct {
			Settings struct {
				Depguard struct {
					Rules map[string]struct {
						Files []string `yaml:"files"`
					} `yaml:"rules"`
				} `yaml:"depguard"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	return config.Linters.Settings.Depguard.Rules["driving-adapter"].Files
}

// getUncovered are the driving adapters of adapter/ no pattern of the lint rule
// reaches.
func getUncovered(patterns []string) []string {
	var found []string
	for one := range driving {
		if !strings.HasPrefix(one, "adapter/") {
			continue
		}
		if !slices.ContainsFunc(patterns, func(pattern string) bool {
			target := strings.TrimSuffix(strings.TrimPrefix(pattern, "**/"), "/**")
			return !strings.HasPrefix(pattern, "!") && (one == target || strings.HasPrefix(one, target+"/"))
		}) {
			found = append(found, one)
		}
	}
	slices.Sort(found)
	return found
}

// Every driving adapter is on the lint rule that keeps it off the ports, with zero ! file exemptions.
func TestEveryDrivingAdapterIsOnTheLintRule(t *testing.T) {
	patterns := depguardFiles(t)
	if len(patterns) == 0 {
		t.Fatal("the driving-adapter rule lists no file: the config is not being read")
	}
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") && p != "!$test" {
			t.Errorf("driving-adapter depguard rule contains file exemption %s: all driving adapters must go through use cases", p)
		}
	}
	for _, one := range getUncovered(patterns) {
		t.Errorf("%s is a driving adapter and the depguard driving-adapter rule does not list it", one)
	}
}

// TestNoDrivingAdapterImportsAPortDirectly verifies that no driving adapter in libs/core or apps/desktop imports a port.
func TestNoDrivingAdapterImportsAPortDirectly(t *testing.T) {
	roots := []string{"../adapter/cli", "../adapter/window", "../internal/adapter/plugin", "../internal/adapter/theme", "../../../apps/desktop"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				val := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(val, module+"port/") || val == module+"port" {
					t.Errorf("%s imports %s: driving adapter must depend on usecase, not port", path, val)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLintCoverageRule_ReportsUncoveredAdapters(t *testing.T) {
	got := getUncovered([]string{"**/adapter/window/**", "!**/adapter/cli/deps.go"})
	if want := []string{"adapter/cli"}; !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
	if got := getUncovered([]string{"**/adapter/window/**", "**/adapter/cli/**"}); len(got) != 0 {
		t.Errorf("the rule refuses %v, want none", got)
	}
}
