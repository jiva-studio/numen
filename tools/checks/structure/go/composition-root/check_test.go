// Package compositionroot is the check `go/composition-root`, run on the tree with the planted violations that prove it.
package compositionroot

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule rejects a package below the root, and accepts the root and a package beside it.
func TestCompositionRootRule_RejectsAPackageBelowTheRoot(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"container/container.go":         "package container\n\nfunc New() {}\n",
		"container/settings/settings.go": "package settings\n\nfunc Load() {}\n",
		"containerkit/kit.go":            "package containerkit\n\nfunc Build() {}\n",
	})
	found, isRootRead := getRootSubpackages(load.LoadFixture(t, dir), "fixture/container")
	if want := []string{"settings.go:1 fixture/container/settings"}; !isRootRead || !slices.Equal(load.FormatFindings(found), want) {
		t.Errorf("the rule rejects %v with the root read %v, want %v", load.FormatFindings(found), isRootRead, want)
	}
}

func TestTheCompositionRootIsOnePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkCompositionRootIsOnePackage(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
