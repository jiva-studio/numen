package container

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// A list that only shrinks has to be made to shrink. An entry naming an edge
// the tree no longer has is a rule written down for nothing: it goes on
// admitting what it names, so the day somebody reaches for that edge again it
// is let through, and nobody decided that.
func TestNothingIsBaselinedThatIsNoLongerReached(t *testing.T) {
	for from, entries := range baseline {
		for _, to := range entries {
			if !isAdmitting(t, from, to) {
				t.Errorf("the baseline holds %s → %s and nothing it reaches is refused", from, to)
			}
		}
	}
}

// isAdmitting says whether one entry of the baseline still lets an edge
// through: the entry is taken off the list, the package's own imports are read,
// and the rule is asked again. An entry that changes no answer is admitting
// nothing.
func isAdmitting(t *testing.T, from, to string) bool {
	t.Helper()

	kept := baseline[from]
	baseline[from] = slices.DeleteFunc(slices.Clone(kept), func(one string) bool { return one == to })
	defer func() { baseline[from] = kept }()

	for _, dep := range getImports(t, from) {
		if dep != to && !strings.HasPrefix(dep, to+"/") {
			continue
		}
		if getRefusal(from, dep) != "" {
			return true
		}
	}
	return false
}

// getImports are the packages of this module one package's own files import. A
// test file is left out for the reason the rules leave it out.
func getImports(t *testing.T, pkg string) []string {
	t.Helper()

	at := filepath.Join("..", filepath.FromSlash(pkg))
	held, err := os.ReadDir(at)
	if err != nil {
		t.Fatal(err)
	}
	var named []string
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(at, one.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			to, err := strconv.Unquote(imported.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			named = append(named, strings.TrimPrefix(to, module))
		}
	}
	return named
}

// What the shrink rule refuses, read against entries written to be refused: one
// naming an edge the package does not have, and one naming an edge it has and
// is not refused for. The entries doing work have to come back admitting, or
// the walk is passing on an empty read.
func TestWhatTheBaselineShrinkRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		from, to   string
		isAdmitted bool
	}{
		{"container", "task", true},
		{"adapter/cli", "internal/adapter/trash", false},
		{"adapter/settings", "port", false},
	} {
		if got := isAdmitting(t, one.from, one.to); got != one.isAdmitted {
			if one.isAdmitted {
				t.Errorf("the baseline entry %s → %s admits nothing", one.from, one.to)
			} else {
				t.Errorf("%s → %s needs no baseline entry and the rule admits one", one.from, one.to)
			}
		}
	}
}
