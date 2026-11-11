// Package windowadaptersession is the check `go/window-adapter-session`, run on the tree with the planted violations that prove it.
package windowadaptersession

import (
	"os"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestNoWindowAdapterBoundsAWait(t *testing.T) {
	var wrong []string
	files, packages := walkWindowAdapters(t, func(_, path string) {
		for _, call := range getWaitFindings(t, path, nil) {
			wrong = append(wrong, path+" calls "+call)
		}
	})
	for _, one := range wrong {
		t.Error(one + ": a bound on a wait is a decision of a use case")
	}

	if files == 0 || packages == 0 {
		t.Fatalf("%d files in %d packages read: the walk is not reading the window adapters", files, packages)
	}
}

func TestEditorWindowImportsNoSync(t *testing.T) {
	var wrong []string
	files, _ := walkWindowAdapters(t, func(pkg, path string) {
		if !load.IsUnder(pkg, "adapter/window/editor") {
			return
		}
		for range getLockImports(t, path, nil) {
			wrong = append(wrong, path+" imports sync")
		}
	})
	for _, one := range wrong {
		t.Error(one + ": what is locked while a window switches vaults belongs to the window session use case")
	}
	if files == 0 {
		t.Fatalf("%d files of the editor window read: the walk is not reading it", files)
	}
}

func TestTheWaitRuleSeesABoundOnAWait(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      int
	}{
		{"timeout", "package p\nimport (\"context\"; \"time\")\nfunc f(ctx context.Context) { context.WithTimeout(ctx, time.Second) }", 1},
		{"deadline", "package p\nimport (\"context\"; \"time\")\nfunc f(ctx context.Context) { context.WithDeadline(ctx, time.Now()) }", 1},
		{"after", "package p\nimport \"time\"\nfunc f() { <-time.After(time.Second) }", 1},
		{"sleep", "package p\nimport \"time\"\nfunc f() { time.Sleep(time.Second) }", 1},
		{"ticker of a stream", "package p\nimport \"time\"\nfunc f() { time.NewTicker(time.Second) }", 0},
		{"cancel without a bound", "package p\nimport \"context\"\nfunc f(ctx context.Context) { context.WithCancel(ctx) }", 0},
	} {
		if got := getWaitFindings(t, tc.name, tc.src); len(got) != tc.want {
			t.Errorf("%s: %d calls found, want %d", tc.name, len(got), tc.want)
		}
	}
}

func TestTheLockRuleSeesTheSyncImport(t *testing.T) {
	if got := getLockImports(t, "a.go", "package p\nimport \"sync\"\nvar mu sync.Mutex"); len(got) != 1 {
		t.Errorf("%d imports of sync found in a file that imports it, want 1", len(got))
	}
	if got := getLockImports(t, "b.go", "package p\nimport \"sync/atomic\"\nvar n atomic.Int32"); len(got) != 0 {
		t.Errorf("%d imports of sync found in a file that imports sync/atomic only, want 0", len(got))
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
