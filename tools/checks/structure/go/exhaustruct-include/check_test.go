// Package exhaustructinclude is the check `go/exhaustruct-include`, run on the tree with the planted violations that prove it.
package exhaustructinclude

import (
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A pattern that matches no type is reported, and one that matches a type is not.
func TestExhaustructIncludeRule_ReportsAPatternThatMatchesNoType(t *testing.T) {
	names := []string{"example.com/a/domain/model.Model", "example.com/a/domain/model.Vector"}
	got := getUnmatchedPatterns([]string{`example\.com/a/domain/model\.Model$`, `example\.com/a/core/port\.Model$`}, names)
	if len(got) != 1 || got[0] != `example\.com/a/core/port\.Model$` {
		t.Errorf("unmatched patterns %v, want only the pattern of the missing package", got)
	}
}

func TestEveryExhaustructIncludeMatchesAType(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkExhaustructIncludesMatchTypes(t, load.Workspace(t))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
