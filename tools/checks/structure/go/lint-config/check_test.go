// Package lintconfig is the check `go/lint-config`, run on the tree with the planted violations that prove it.
package lintconfig

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

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

func TestMutationDirectiveRule_RejectsMissingReasons(t *testing.T) {
	root, err := filepath.Abs(load.RepoRootDir)
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

func TestEveryExhaustructPatternMatchesAType(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkEveryExhaustructPatternMatchesAType(t, load.Workspace(t))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
