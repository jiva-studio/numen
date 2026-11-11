package editor

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

// A change arrives at every subscriber drawing the vault, the same way a note
// opened in the editor does.
func TestAChangeReachesEveryoneDrawing(t *testing.T) {
	state := NewState(run.NewTasks(), nil, nil)
	line, isDone := state.Notices.Edits.subscribe()
	t.Cleanup(isDone)

	response := domain.Edit{
		Change: "one", Path: "Aggressor.md", From: 2, To: 12, Text: "An axe",
	}
	if err := ViewOf(state).ShowEdit(t.Context(), response); err != nil {
		t.Fatal(err)
	}

	if got := <-line; got != response {
		t.Errorf("what arrived is %+v", got)
	}
}

// Every report carries the whole of what a change is doing, so a listener that
// has not read one is given the newest and not the backlog.
func TestTheLastChangeGeneratedIsTheOneWaiting(t *testing.T) {
	broadcaster := newEditBroadcaster()
	line, isDone := broadcaster.subscribe()
	t.Cleanup(isDone)

	broadcaster.publish(domain.Edit{Change: "one", Text: "first"})
	broadcaster.publish(domain.Edit{Change: "one", Text: "second"})

	if got := <-line; got.Text != "second" {
		t.Errorf("what waited is %q", got.Text)
	}
}

// The report that ends a change is not one a listener may be spared: a drawing
// that is never ended stays on the screen.
func TestTheReportThatEndsAChangeIsNotReplaced(t *testing.T) {
	broadcaster := newEditBroadcaster()
	line, isDone := broadcaster.subscribe()
	t.Cleanup(isDone)

	broadcaster.publish(domain.Edit{Change: "one", IsDone: true})
	broadcaster.publish(domain.Edit{Change: "one", Text: "more of it"})

	got := <-line
	if got.Change != "one" || !got.IsDone {
		t.Errorf("what waited is %+v, want the report that ended change one", got)
	}
	// What could not replace it waits behind it.
	if got := <-line; got.Text != "more of it" {
		t.Errorf("what waited behind it is %+v, want the report of change one", got)
	}
}

// Several changes are made at once, and each is drawn in a place of its own: a
// report about one note may not take the place of an unread report about
// another.
func TestAChangeDoesNotDisplaceOneToAnotherNote(t *testing.T) {
	broadcaster := newEditBroadcaster()
	line, isDone := broadcaster.subscribe()
	t.Cleanup(isDone)

	broadcaster.publish(domain.Edit{Change: "one", Path: "Aggressor.md", Text: "An axe"})
	broadcaster.publish(domain.Edit{Change: "two", Path: "Fugue.md", Text: "A theme"})
	broadcaster.publish(domain.Edit{Change: "two", Path: "Fugue.md", Text: "A theme answered"})

	if got := <-line; got.Change != "one" || got.Text != "An axe" {
		t.Errorf("the first report waiting is %+v, want change one saying %q", got, "An axe")
	}
	if got := <-line; got.Change != "two" || got.Text != "A theme answered" {
		t.Errorf("the next is %+v, want change two saying %q", got, "A theme answered")
	}
	select {
	case extra := <-line:
		t.Errorf("a report was queued behind them: %+v", extra)
	default:
	}
}

// An error code is about the note the request named. Only what the core named
// as missing is answered as one.
func TestOnlyTheCoreReturnsWhichNoteIsMissing(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.ErrorCode
		is   bool
	}{
		"the vault holds no note there": {
			err:  fmt.Errorf("read Old.md: %w", edit.ErrNoNote),
			want: v1.ErrorCode_ERROR_CODE_NOT_FOUND,
			is:   true,
		},
		"the vault itself is out of reach": {
			err: fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
		},
		"a title no note can be given": {
			err:  fmt.Errorf("%w: nothing to name it", domain.ErrInvalidName),
			want: v1.ErrorCode_ERROR_CODE_INVALID_NAME,
			is:   true,
		},
		"a frontmatter written on one line": {
			err:  fmt.Errorf("Old.md: %w", edit.ErrInline),
			want: v1.ErrorCode_ERROR_CODE_UNREADABLE,
			is:   true,
		},
		"a frontmatter block that is never closed": {
			err:  fmt.Errorf("Old.md: %w", edit.ErrUnterminated),
			want: v1.ErrorCode_ERROR_CODE_UNREADABLE,
			is:   true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reason, rejected := wire.GetErrorCode(c.err)
			if rejected != c.is {
				t.Fatalf("want rejected=%v, got %v (%v)", c.is, rejected, reason)
			}
			if reason != c.want {
				t.Errorf("want %v, got %v", c.want, reason)
			}
		})
	}
}

func levelNothing(context.Context, domain.Vault, []string) error { return nil }

func openEditor(t *testing.T, notes map[string]string) *Services {
	t.Helper()
	writing := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, levelNothing, testsupport.NewClock())
	read := note.NewRead(vaultreader.Readers{})
	api := newBareServices()
	api.NoteCases.Read, api.NoteCases.Write = &read, &writing
	api.UseNotes()
	api.state.Runtime.SetVault(vaultfixture.NewVault(t, notes))
	return api
}

func readNote(t *testing.T, api *Services, path string) *v1.ReadBaseline {
	t.Helper()
	out, err := api.Note.ReadNote(t.Context(), connect.NewRequest(&v1.ReadNoteRequest{Path: path}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetAt() == nil {
		t.Fatalf("the read of %s carried no fingerprint", path)
	}
	return &v1.ReadBaseline{Prose: out.Msg.GetBody(), At: out.Msg.GetAt()}
}

// A note holding prose the client has not read is its own answer.
func TestWriteNote_ReportsConflictOnUnreadDiskChange(t *testing.T) {
	api := openEditor(t, map[string]string{"Entropy.md": "# Entropy\n"})
	baseline := readNote(t, api, "Entropy.md")

	theirs := "# Entropy\n\nTheirs.\n"
	on := filepath.Join(api.state.GetCurrentVault().Path, "Entropy.md")
	if err := os.WriteFile(on, []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := api.Note.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path:     "Entropy.md",
		Body:     "# Entropy\n\nMine.\n",
		Baseline: baseline,
	}))
	if err != nil {
		t.Fatalf("a note that changed came back as an error: %v", err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_CONFLICT {
		t.Errorf("a note that changed was answered %v", code)
	}
	if out.Msg.GetAt() != nil {
		t.Error("a write that wrote nothing answered with a fingerprint")
	}

	raw, err := os.ReadFile(on)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != theirs {
		t.Errorf("a write that was stopped wrote anyway:\n%s", raw)
	}
}

// The file stat a write answers with is what the client presents at its next
// write, and it is what lets a session hold more than one save.
func TestAWriteAnswersWithTheFileItProduced(t *testing.T) {
	api := openEditor(t, map[string]string{"Entropy.md": "# Entropy\n"})
	baseline := readNote(t, api, "Entropy.md")

	first, err := api.Note.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path:     "Entropy.md",
		Body:     "# Entropy\n\nOne.\n",
		Baseline: baseline,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetError() == v1.ErrorCode_ERROR_CODE_CONFLICT {
		t.Fatal("the first write said the note changed")
	}
	if first.Msg.GetAt() == nil {
		t.Fatal("the write answered with no fingerprint")
	}

	second, err := api.Note.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path:     "Entropy.md",
		Body:     "# Entropy\n\nTwo.\n",
		Baseline: &v1.ReadBaseline{Prose: baseline.GetProse(), At: first.Msg.GetAt()},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.GetError() == v1.ErrorCode_ERROR_CODE_CONFLICT {
		t.Error("the write after a write said the note changed")
	}

	raw, err := os.ReadFile(filepath.Join(api.state.GetCurrentVault().Path, "Entropy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "# Entropy\n\nTwo.\n" {
		t.Errorf("the second write did not land:\n%s", raw)
	}
}

// beaten is a vault's readers with something writing the note the moment it has
// been looked at, which is what a writer outside this process looks like from
// in here.
type beaten struct {
	vaultport.VaultReaders
	after func(path string)
}

func (b beaten) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := b.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return beatenReader{VaultReader: reader, after: b.after}, nil
}

type beatenReader struct {
	vaultport.VaultReader
	after func(path string)
}

func (b beatenReader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	ref, err := b.VaultReader.StatFile(ctx, path)
	b.after(path)
	return ref, err
}

// A join reads a note, splices its frontmatter and puts it back.
func TestJoinNote_ReportsConflictWhenNoteChanged(t *testing.T) {
	api := openEditor(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})
	on := filepath.Join(api.state.GetCurrentVault().Path, "Heat.md")

	var once sync.Once
	linking := note.NewEditLinks(
		beaten{VaultReaders: vaultreader.Readers{}, after: func(path string) {
			if path != "Heat.md" {
				return
			}
			once.Do(func() {
				if err := os.WriteFile(on, []byte("# Heat\n\nSomebody else's, longer.\n"), 0o644); err != nil {
					t.Error(err)
				}
			})
		}},
		vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()},
		levelNothing,
		testsupport.NewClock(),
	)
	api.NoteCases.EditLinks = &linking
	api.UseNotes()

	out, err := api.Note.WriteLink(t.Context(), connect.NewRequest(&v1.WriteLinkRequest{
		Path: "Heat.md",
		Link: &v1.Link{To: "Entropy.md", Role: v1.Role_ROLE_PARENT},
	}))
	if err != nil {
		t.Fatalf("a note that changed came back as an error: %v", err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_CONFLICT {
		t.Errorf("a note that changed was answered %v", code)
	}

	raw, err := os.ReadFile(on)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "to: Entropy") {
		t.Errorf("a join that was stopped wrote anyway:\n%s", raw)
	}
}

// TestEditLoad measures the load path.
func TestEditLoad(t *testing.T) {
	if os.Getenv("NUMEN_LOAD") == "" {
		t.Skip("set NUMEN_LOAD=1 to run the load test")
	}
	notes := 100_000
	if s := os.Getenv("NUMEN_LOAD_NOTES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("NUMEN_LOAD_NOTES: %v", err)
		}
		notes = n
	}

	v := vaultfixture.GenerateVault(t, notes)
	wired := newWiring(filepath.Join(t.TempDir(), "index.db"), false, nil)
	db := wired.openIndex(t)

	writing := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, levelNothing, testsupport.NewClock())
	read := note.NewRead(vaultreader.Readers{})
	api := createServices(NewState(run.NewTasks(), nil, nil))
	bindNotes(api, db)
	api.NoteCases.Read, api.NoteCases.Write = &read, &writing
	api.UseNotes()
	api.state.Indexing.Embedding = indexing.NewEmbedding(nil, nil, db.Chunks.Progress)
	api.state.Runtime.SetVault(v)

	startTime := testsupport.GetMachine().Now()
	end := wired.startSession(t, t.Context(), db, api.state, v, wired.reading.GetVaultReaders(), wired.reading.GetVaultWatcher(), sessionOptions{
		idle: time.Hour, out: io.Discard,
	})
	t.Cleanup(end)
	for !api.state.Runtime.IsReady() && api.state.Runtime.GetError() == "" {
		time.Sleep(50 * time.Millisecond)
	}
	if why := api.state.Runtime.GetError(); why != "" {
		t.Fatalf("the vault could not be read: %s", why)
	}
	t.Logf("read %d notes in %s", notes, testsupport.GetMachine().Now().Sub(startTime).Round(time.Millisecond))

	line, isDone := api.state.Notices.ChangeEvents.subscribe()
	t.Cleanup(isDone)

	const path = "01/note-000001.md"
	body := "# Heat\n\nA line nobody wrote before, at " + strconv.FormatInt(testsupport.GetMachine().Now().UnixNano(), 10) + ".\n"

	saving := testsupport.GetMachine().Now()
	if _, err := api.NoteCases.Write.Save(t.Context(), v, path, body, nil); err != nil {
		t.Fatal(err)
	}
	wrote := testsupport.GetMachine().Now().Sub(saving)

	durationCh := make(chan time.Duration, 1)
	go func() {
		for what := range line {
			if slices.Contains(what.Paths, path) {
				durationCh <- testsupport.GetMachine().Now().Sub(saving)
				return
			}
			if what.ShouldReload {
				durationCh <- testsupport.GetMachine().Now().Sub(saving)
				return
			}
		}
	}()

	select {
	case reached := <-durationCh:
		t.Logf("save %s, the window told after %s",
			wrote.Round(time.Millisecond), reached.Round(time.Millisecond))
	case <-time.After(2 * time.Minute):
		t.Fatalf("the change never reached a client; the save itself took %s", wrote)
	}
}
