// Package containerdowncast is the check `go/container-downcast`, run on the tree with the planted violations that prove it.
package containerdowncast

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The container receives concrete plugin capabilities during assembly.
func TestContainerDoesNotDowncastPluginHost(t *testing.T) {
	wrong, read, err := findPluginDowncasts(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Errorf("%s requires a concrete plugin type assertion; pass the required role during assembly", one)
	}
	if read == 0 {
		t.Fatal("no file of the container read: the rule is checked against nothing")
	}
}

// The rule rejects an assertion to a pointer of package plugin, and passes an
// assertion to an interface of package plugin, to a pointer of another package and one in a test file.
func TestDowncastRule_RejectsAssertionsToPluginRoles(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"host.go": `package container

func roles(host any) {
	_ = host.(*plugin.Discovery)
	_ = host.(plugin.Host)
	_ = host.(*catalog.Entry)
}
`,
		"host_test.go": `package container

func check(host any) { _ = host.(*plugin.Discovery) }
`,
	})
	wrong, read, err := findPluginDowncasts(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"host.go: assertion to *plugin.Discovery"}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 1 {
		t.Errorf("%d files read, want 1", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
