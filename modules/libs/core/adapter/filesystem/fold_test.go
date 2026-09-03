package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
)

// event is one change, as the operating system hands it over.
type event struct{ path string }

func (e event) Event() notify.Event { return notify.Write }
func (e event) Path() string        { return e.path }
func (e event) Sys() any            { return nil }

// TestABacklogThatStaysFullMeansTheVaultIsReadAgain. Past the end of the
// backlog the operating system drops events and says nothing, so what was
// missed cannot be worked out and the vault is walked again.
func TestABacklogThatStaysFullMeansTheVaultIsReadAgain(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := Open(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	raw := make(chan notify.EventInfo, 4)
	changes := make(chan []string)
	lost := make(chan struct{}, 1)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go fold(ctx, remembered(reader), Options{}, raw, changes, lost)

	one := event{path: filepath.Join(root, "Note.md")}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw <- one:
			}
		}
	}()

	select {
	case <-lost:
	case <-time.After(5 * time.Second):
		t.Fatal("a backlog that never emptied was not reported as lost")
	}
}
