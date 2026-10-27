package editor

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// A change reaches everyone drawing the vault the way a note put in front of
// the person does.
func TestAChangeReachesEveryoneDrawing(t *testing.T) {
	state := &State{Edits: newEditBroadcaster()}
	line, isDone := state.Edits.subscribe()
	t.Cleanup(isDone)

	response := domain.Edit{
		Change: "one", Path: "Aggressor.md", From: 2, To: 12, Text: "An axe",
	}
	if err := state.GetWindow().ShowEdit(t.Context(), response); err != nil {
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
			err:  fmt.Errorf("read Old.md: %w", note.ErrNoNote),
			want: v1.ErrorCode_ERROR_CODE_MISSING,
			is:   true,
		},
		"the vault itself is out of reach": {
			err: fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
		},
		"a title no note can be given": {
			err:  fmt.Errorf("%w: nothing to name it", note.ErrUnnameable),
			want: v1.ErrorCode_ERROR_CODE_UNNAMEABLE,
			is:   true,
		},
		"a frontmatter written on one line": {
			err:  fmt.Errorf("Old.md: %w", note.ErrInline),
			want: v1.ErrorCode_ERROR_CODE_UNREADABLE,
			is:   true,
		},
		"a frontmatter block that is never closed": {
			err:  fmt.Errorf("Old.md: %w", note.ErrUnterminated),
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

// levelNothing brings nothing level: what is asked in these tests is what the
// schema carries, and there is no index behind them to find it in.
func levelNothing(context.Context, domain.Vault, []string) error { return nil }

// openEditor is a vault with a read and a save on it and nothing behind them.
func openEditor(t *testing.T, notes map[string]string) *Services {
	t.Helper()
	writing := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, levelNothing, time.Now)
	read := note.NewRead(vaultreader.Readers{})
	api := newBareServices()
	api.Note.read, api.Note.write = &read, &writing
	api.state.setVault(testsupport.NewVault(t, notes))
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
func TestAWriteOverProseTheClientNeverReadIsAnsweredChanged(t *testing.T) {
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
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_STALE {
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

// The fingerprint a write answers with is what the client presents at its next
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
	if first.Msg.GetError() == v1.ErrorCode_ERROR_CODE_STALE {
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
	if second.Msg.GetError() == v1.ErrorCode_ERROR_CODE_STALE {
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

func (b beatenReader) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	ref, err := b.VaultReader.GetFingerprint(ctx, path)
	b.after(path)
	return ref, err
}

// A join reads a note, splices its frontmatter and puts it back.
func TestAJoinOverANoteThatMovedIsAnsweredChanged(t *testing.T) {
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
		vaultwriter.VaultWriters{},
		levelNothing,
		time.Now,
	)
	api.Note.editLinks = &linking

	out, err := api.Note.WriteLink(t.Context(), connect.NewRequest(&v1.WriteLinkRequest{
		Path: "Heat.md",
		Link: &v1.Link{To: "Entropy.md", Role: v1.Role_ROLE_PARENT},
	}))
	if err != nil {
		t.Fatalf("a note that changed came back as an error: %v", err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_STALE {
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

	v := testsupport.GenerateVault(t, notes)
	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	writing := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, levelNothing, time.Now)
	read := note.NewRead(vaultreader.Readers{})
	api := createServices(&State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
	})
	bindNotes(api, db)
	api.Note.read, api.Note.write = &read, &writing
	api.state.Indexing.Progress = db.Progress()
	api.state.setVault(v)

	startTime := time.Now()
	made := container.NewEditorAssembly(cfg, nil, db)
	worker := newIndexWorker(made, api.state, nil, time.Hour, io.Discard)
	_, wait := startIndexing(t.Context(), v, made, api.state, false, worker, worker.NewSession(), io.Discard)
	t.Cleanup(wait)
	for !api.state.Ready.Load() && api.state.Error.Why() == "" {
		time.Sleep(50 * time.Millisecond)
	}
	if why := api.state.Error.Why(); why != "" {
		t.Fatalf("the vault could not be read: %s", why)
	}
	t.Logf("read %d notes in %s", notes, time.Since(startTime).Round(time.Millisecond))

	line, isDone := api.state.ChangeEvents.subscribe()
	t.Cleanup(isDone)

	const path = "01/note-000001.md"
	body := "# Heat\n\nA line nobody wrote before, at " + strconv.FormatInt(time.Now().UnixNano(), 10) + ".\n"

	saving := time.Now()
	if _, err := api.Note.write.Save(t.Context(), v, path, body, nil); err != nil {
		t.Fatal(err)
	}
	wrote := time.Since(saving)

	durationCh := make(chan time.Duration, 1)
	go func() {
		for what := range line {
			for _, p := range what.paths {
				if p == path {
					durationCh <- time.Since(saving)
					return
				}
			}
			if what.shouldReload {
				durationCh <- time.Since(saving)
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

// The window says what currently open, and whoever answers on their behalf
// reads the last of it.

func TestAttendingIsWhatTheWindowLastSaid(t *testing.T) {
	state := &State{}
	api := &workspaceHandler{state: state}

	if open := state.GetOpenTabs(); len(open.Tabs) != 0 || open.FrontID != "" {
		t.Fatalf("a window that has said nothing has %+v open", open)
	}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		Front: "two",
		Tabs: []*v1.Tab{
			{Id: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{Id: "two", Kind: "reader:player", Path: "Talk.bin", Title: "Talk", State: `{"t":[12]}`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := domain.OpenTabs{
		FrontID: "two",
		Tabs: []domain.Tab{
			{ID: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{ID: "two", Kind: "reader:player", Path: "Talk.bin", Title: "Talk", State: `{"t":[12]}`},
		},
	}
	if got := state.GetOpenTabs(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window has %+v open", got)
	}
}

// memoryTabs keeps the open tabs of each vault in memory.
type memoryTabs map[domain.VaultID]domain.OpenTabs

func (m memoryTabs) ReadOpenTabs(_ context.Context, v domain.Vault) (domain.OpenTabs, error) {
	return m[v.ID], nil
}

func (m memoryTabs) WriteOpenTabs(_ context.Context, v domain.Vault, open domain.OpenTabs) error {
	m[v.ID] = open
	return nil
}

func TestReadOpenTabsReturnsTabsSavedForVault(t *testing.T) {
	store := memoryTabs{}
	state := &State{OpenTabsStore: store}
	state.setVault(domain.Vault{ID: "v", Path: t.TempDir()})
	api := &workspaceHandler{state: state}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		Front: "one",
		Tabs:  []*v1.Tab{{Id: "one", Kind: "reader:viewer", Path: "a.bin", Title: "a", State: `{"page":3}`}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	// A core started again reads the store, not the tabs held in memory.
	restarted := &workspaceHandler{state: &State{OpenTabsStore: store}}
	restarted.state.setVault(state.GetCurrentVault())
	got, err := restarted.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.ReadOpenTabsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetFront() != "one" || len(got.Msg.GetTabs()) != 1 || got.Msg.GetTabs()[0].GetState() != `{"page":3}` {
		t.Errorf("read %+v", got.Msg)
	}
}

func TestReadOpenTabsReturnsNoneWithoutAVault(t *testing.T) {
	api := &workspaceHandler{state: &State{OpenTabsStore: memoryTabs{}}}
	got, err := api.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.ReadOpenTabsRequest{}))
	if err != nil || len(got.Msg.GetTabs()) != 0 {
		t.Errorf("read %+v, %v", got.Msg, err)
	}
}

func TestAttendingTellsWhoeverIsListening(t *testing.T) {
	var received []domain.OpenTabs
	state := &State{NotifyOpenTabs: func(open domain.OpenTabs) { received = append(received, open) }}
	api := &workspaceHandler{state: state}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		Front: "one",
		Tabs:  []*v1.Tab{{Id: "one", Kind: "note", Path: "Entropy.md", Title: "Entropy"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(received) != 1 {
		t.Fatalf("what the window has open was said %d times", len(received))
	}
	front, target := received[0].GetFrontTab()
	if !target || front.Path != "Entropy.md" {
		t.Errorf("the tab in front is %+v", front)
	}
}

func TestFrontedIsTheTabThePersonIsLookingAt(t *testing.T) {
	for name, c := range map[string]struct {
		open domain.OpenTabs
		want string
	}{
		"a tab in front": {
			open: domain.OpenTabs{
				FrontID: "two",
				Tabs:    []domain.Tab{{ID: "one", Path: "Entropy.md"}, {ID: "two", Path: "Talk.mp3"}},
			},
			want: "Talk.mp3",
		},
		"nothing open":     {open: domain.OpenTabs{}},
		"a tab it lets go": {open: domain.OpenTabs{FrontID: "three", Tabs: []domain.Tab{{ID: "one"}}}},
		"a tab with no identity": {
			open: domain.OpenTabs{Tabs: []domain.Tab{{Path: "Entropy.md"}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			front, target := c.open.GetFrontTab()
			if target != (c.want != "") {
				t.Fatalf("a tab in front is %v", target)
			}
			if front.Path != c.want {
				t.Errorf("the tab in front holds %q", front.Path)
			}
		})
	}
}

func TestDetachTabInvokesState(t *testing.T) {
	var calledTab string
	var calledX, calledY int
	state := &State{
		DetachTab: func(_ context.Context, tab string, x, y int) error {
			calledTab = tab
			calledX = x
			calledY = y
			return nil
		},
	}
	api := &workspaceHandler{state: state}

	_, err := api.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{
		TabId: "tab-123",
		X:     150,
		Y:     250,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if calledTab != "tab-123" || calledX != 150 || calledY != 250 {
		t.Errorf("detached tab: got (%q, %d, %d), want (%q, 150, 250)", calledTab, calledX, calledY, "tab-123")
	}
}

func TestDetachTabFailsWhenUnimplemented(t *testing.T) {
	state := &State{}
	api := &workspaceHandler{state: state}

	_, err := api.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{
		TabId: "tab-123",
	}))
	if err == nil {
		t.Fatal("expected error when DetachTab is nil, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("got code %v, want %v", connect.CodeOf(err), connect.CodeUnavailable)
	}
}
