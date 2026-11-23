// Package structcontext is the check `go/struct-context`, run on the tree with the planted violations that prove it.
package structcontext

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestNoStructCarriesAContext verifies that no struct field across Go modules holds context.Context.
func TestNoStructCarriesAContext(t *testing.T) {
	wrong, read, err := findContextFields([]string{"..", "../../../shared/note", "../../../plugin/sdk-go", "../../../../plugins", "../../../../apps/desktop"})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if read == 0 {
		t.Fatal("no source file read: the rule is checked against nothing")
	}
}

// The rule rejects a named and an embedded context field, one in a struct declared inside a function included,
// and passes a context parameter, a test file and a generated protocol file.
func TestStructContextRule_RejectsContextFields(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"job/job.go": `package job

import "context"

type Job struct {
	ctx  context.Context
	Name string
}

type Wrapped struct {
	context.Context
}

func Run(ctx context.Context) {
	type local struct{ parent context.Context }
	_ = local{parent: ctx}
}
`,
		"job/job_test.go": `package job

import "context"

type fake struct{ ctx context.Context }
`,
		"gen/job.pb.go": `package gen

import "context"

type Message struct{ ctx context.Context }
`,
	})
	wrong, read, err := findContextFields([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "job", "job.go")
	const reason = " holds context.Context in a struct field (pass context per method call instead)"
	want := []string{path + " Job.ctx" + reason, path + " Wrapped.<embedded>" + reason, path + " local.parent" + reason}
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
