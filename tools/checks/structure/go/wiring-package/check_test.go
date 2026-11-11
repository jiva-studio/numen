// Package wiringpackage is the check `go/wiring-package`, run on the tree with the planted violations that prove it.
package wiringpackage

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule rejects a package that imports a use case and a driven adapter, and accepts a driving adapter,
// a driven adapter, a package that imports one of the two, and the exempt composition root.
func TestWiringPackageRule_RejectsAPackageThatBuildsUseCasesFromDrivenAdapters(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": "package noteport\n\ntype Notes interface{ Read() string }\n",
		"usecase/find/find.go": `package find

import "fixture/port/noteport"

type Find struct{ notes noteport.Notes }

func New(notes noteport.Notes) *Find { return &Find{notes: notes} }
`,
		"adapter/store/store.go": "package store\n\ntype Store struct{}\n\nfunc (*Store) Read() string { return \"\" }\n",
		"container/container.go": `package container

import (
	"fixture/adapter/store"
	"fixture/usecase/find"
)

func New() *find.Find { return find.New(&store.Store{}) }
`,
		"internal/hub/hub.go": `package hub

import (
	"fixture/adapter/store"
	"fixture/usecase/find"
)

func Build() *find.Find { return find.New(&store.Store{}) }
`,
		"adapter/driving/driving.go": `package driving

import "fixture/usecase/find"

func Run(f *find.Find) {}
`,
		"internal/stores/stores.go": `package stores

import "fixture/adapter/store"

func Open() *store.Store { return &store.Store{} }
`,
	})
	found, exempted := getWiringPackages(load.LoadFixture(t, dir), "fixture", func(path string) bool { return path == "fixture/container" })
	want := []string{"hub.go:1 fixture/internal/hub"}
	if got := load.FormatFindings(found); !slices.Equal(got, want) || exempted != 1 {
		t.Errorf("the rule rejects %v with %d exempt, want %v with 1", got, exempted, want)
	}
}

func TestCompositionRootRule_ExemptsTheContainerAndTheApplicationRoots(t *testing.T) {
	for _, tc := range []struct {
		path     string
		isExempt bool
	}{
		{load.CoreImportPath + "/container", true},
		{load.CoreImportPath + "/container/vault", true},
		{load.NumenRoot + "apps/desktop/cmd/numen", true},
		{load.NumenRoot + "apps/desktop/app", true},
		{load.NumenRoot + "apps/mobile/app/sub", true},
		{load.NumenRoot + "apps/desktop/internal/platform", false},
		{load.CoreImportPath + "/internal/adapterlib/wire", false},
		{load.CoreImportPath + "/adapter/window/editor", false},
		{load.NumenRoot + "apps/desktop/application", false},
	} {
		if got := isCompositionRoot(tc.path); got != tc.isExempt {
			t.Errorf("%s is exempt: %v, want %v", tc.path, got, tc.isExempt)
		}
	}
}

func TestNoWiringPackageExistsOutsideTheContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoWiringPackageExistsOutsideTheContainer(t, load.Workspace(t))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
