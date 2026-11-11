// Package pluginhostsize is the check `go/plugin-host-size`, run on the tree with the planted violations that prove it.
package pluginhostsize

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestPluginAdapterSizeRule_RejectsAStructOverTheLimits(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"internal/adapter/plugin/host.go": `package plugin

type Many struct{ a, b, c int }

func (Many) One()   {}
func (Many) Two()   {}
func (Many) Three() {}

type Wide struct{ a, b, c, d int }

func (Wide) One() {}

type Small struct{ a int }

func (Small) One() {}
`,
		"internal/adapter/other/other.go": `package other

type Big struct{ a, b, c, d, e int }

func (Big) One() {}
`,
	})
	found, count := getOversizedStructs(load.LoadFixture(t, dir), 2, 3)
	want := []string{
		"host.go:3 plugin.Many has 3 methods and 3 fields",
		"host.go:9 plugin.Wide has 1 methods and 4 fields",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if count.structs != 3 || count.mostMethods != 3 || count.mostFields != 4 {
		t.Errorf("read %+v, want 3 structs of the plugin package, 3 methods and 4 fields at most", count)
	}
}

func TestThePluginAdapterHasNoGodStruct(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkThePluginAdapterHasNoGodStruct(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
