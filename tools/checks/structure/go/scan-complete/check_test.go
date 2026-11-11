// Package scancomplete is the check `go/scan-complete`, run on the tree with the planted violations that prove it.
package scancomplete

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The layers are the folders domain, port, usecase, adapter, internal/adapter, internal/adapterlib,
// internal/testsupport and container. A package belongs to exactly one, so a new folder cannot sit outside
// every rule that reads a layer. The pure packages that belong to none are listed with the reason each needs
// no layer: they hold no scenario and no port, they compute over values or hand out a grant.
func TestEveryKernelPackageBelongsToOneKnownLayer(t *testing.T) {
	dirs, err := listProductionDirs("..")
	if err != nil {
		t.Fatal(err)
	}
	helpers := getHelperPackages()
	for _, one := range getUnplacedPackages(dirs, helpers) {
		t.Error(one)
	}

	// Every helper must exist, or its entry hides a package that is gone and may return unseen.
	for helper, reason := range helpers {
		if !slices.Contains(dirs, helper) || reason == "" {
			t.Errorf("the helper %s is not a package of the kernel or has no reason", helper)
		}
	}
	if len(dirs) < 100 {
		t.Fatalf("%d packages read: the walk is not reading the kernel", len(dirs))
	}
}

// A package outside every group fails, a package in a helper entry passes, and adapter and adapterlib are
// separate groups although one name is the prefix of the other.
func TestScanRule_RejectsAPackageOutsideEveryGroup(t *testing.T) {
	helpers := map[string]string{"internal/csp": "policy string"}
	got := getUnplacedPackages([]string{
		"domain/theme", "port", "port/noteport", "usecase/note/query", "adapter/window/editor", "internal/adapter/clock",
		"internal/adapterlib/wire", "internal/testsupport/indexfake", "container", "internal/csp",
		"internal/vaultreading", "internal/adapterx", "scratch", "internal/csp/sub",
	}, helpers)
	want := []string{
		`internal/vaultreading is in "" groups, want exactly one`,
		`internal/adapterx is in "" groups, want exactly one`,
		`scratch is in "" groups, want exactly one`,
		`internal/csp/sub is in "" groups, want exactly one`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestUseCaseImportRule_RejectsASharedPackageOutsideDomainPortAndHelpers(t *testing.T) {
	helpers := map[string]string{"internal/text": "ranges"}
	got := getBadUseCaseImports(map[string][]string{
		"usecase/note": {"domain/theme", "port/noteport", "internal/text", "internal/vaultreading", "adapter/index", "usecase/note/query"},
	}, helpers)
	want := []string{
		"usecase/note imports adapter/index (adapter): a use case imports the domain, ports and its own helpers",
		`usecase/note imports internal/vaultreading (): a use case imports the domain, ports and its own helpers`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestUseCasesImportOnlyDomainAndPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkUseCasesImportOnlyDomainAndPorts(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
