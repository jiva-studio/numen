package conflict_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/conflict"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const (
	vaultID   domain.VaultID = "Copy"
	list                     = "Shopping list.md"
	listCopy                 = "Shopping list (conflict 2026-11-23).md"
	listCopy2                = "Shopping list (conflict 2026-11-23 2).md"
	pdf                      = "Papers/Report.pdf"
)

type setup struct {
	root *hosttest.Vault
	host ports.Host
	logs *bytes.Buffer
	res  *conflict.Resolver
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	root := hosttest.New(string(vaultID))
	logs := &bytes.Buffer{}
	h := synctest.Host{Root: root}
	return &setup{
		root: root, host: h, logs: logs,
		res: conflict.New(conflict.Deps{Host: h, Log: slog.New(slog.NewTextHandler(logs, nil))}),
	}
}

// input makes the conflict of a path whose local text is already in the vault, and whose head, once replaced, holds
// the text. replaced counts the calls of Replace.
func (s *setup) input(t *testing.T, path, headText string, replaced *int) ports.ConflictInput {
	t.Helper()
	stat, err := s.host.Files(vaultID).StatFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ConflictInput{
		Vault: vaultID, Path: path, Local: ports.FileSide{Hash: stat.Hash, Stat: stat},
		Head: ports.HeadSide{
			Hash: note.Hash([]byte(headText)), Size: int64(len(headText)),
			Replace: func(ctx context.Context, baseline domain.FileStat) error {
				*replaced++
				_, err := s.host.Files(vaultID).Write(ctx, path, []byte(headText), baseline)
				return err
			},
		},
		Day: synctest.GetStart(),
	}
}

func (s *setup) text(path string) (string, bool) { return s.root.Content(path) }

func TestResolve_WritesTheHeadOverTheNoteAndKeepsTheLocalTextInACopy(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	var replaced int

	outcome, err := s.res.Resolve(t.Context(), s.input(t, list, "bread", &replaced))

	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.text(list); got != "bread" {
		t.Errorf("note = %q, want the head bread", got)
	}
	if got, ok := s.text(listCopy); !ok || got != "eggs" {
		t.Errorf("copy = %q (there: %v), want the local text eggs", got, ok)
	}
	if outcome.Copy != listCopy || !slices.Contains(outcome.Applied, list) || !slices.Contains(outcome.Applied, listCopy) {
		t.Errorf("outcome = %+v, want the copy and the paths of both files written", outcome)
	}
}

func TestResolve_WritesTheCopyBeforeTheHeadReplacesTheNote(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	in := s.input(t, list, "bread", new(int))
	replace := in.Head.Replace
	var copyBefore string
	in.Head.Replace = func(ctx context.Context, baseline domain.FileStat) error {
		copyBefore, _ = s.text(listCopy)
		return replace(ctx, baseline)
	}

	if _, err := s.res.Resolve(t.Context(), in); err != nil {
		t.Fatal(err)
	}

	if copyBefore != "eggs" {
		t.Errorf("copy when the head replaced the note = %q, want the local text already kept", copyBefore)
	}
}

func TestResolve_ReplacesTheNoteWithTheBaselineOfTheLocalStat(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	in := s.input(t, list, "bread", new(int))
	var baseline domain.FileStat
	replace := in.Head.Replace
	in.Head.Replace = func(ctx context.Context, got domain.FileStat) error {
		baseline = got
		return replace(ctx, got)
	}

	if _, err := s.res.Resolve(t.Context(), in); err != nil {
		t.Fatal(err)
	}

	if baseline != in.Local.Stat {
		t.Errorf("baseline = %+v, want the stat of the local note %+v", baseline, in.Local.Stat)
	}
}

func TestResolve_AbortsWhenTheCopyCannotBeWritten(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	s.root.RefuseWriting(listCopy, host.ErrStale)
	var replaced int

	_, err := s.res.Resolve(t.Context(), s.input(t, list, "bread", &replaced))

	if err == nil {
		t.Fatal("Resolve = nil, want the error of the copy")
	}
	if got, _ := s.text(list); got != "eggs" || replaced != 0 {
		t.Errorf("note = %q after %d replacements, want the local text kept and the head not written", got, replaced)
	}
}

func TestResolve_NumbersTheCopyWhenTheNameIsTakenByOtherText(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	s.root.File(listCopy, "milk")

	outcome, err := s.res.Resolve(t.Context(), s.input(t, list, "bread", new(int)))

	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.text(listCopy); got != "milk" {
		t.Errorf("first copy = %q, want milk untouched", got)
	}
	if got, _ := s.text(listCopy2); got != "eggs" || outcome.Copy != listCopy2 {
		t.Errorf("second copy = %q at %q, want eggs at %q", got, outcome.Copy, listCopy2)
	}
}

func TestResolve_ReusesACopyOfTheSameTextAfterACrash(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	s.root.File(listCopy, "eggs")

	outcome, err := s.res.Resolve(t.Context(), s.input(t, list, "bread", new(int)))

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.text(listCopy2); ok {
		t.Error("a second copy of the same text was made")
	}
	if got, _ := s.text(list); got != "bread" || outcome.Copy != listCopy {
		t.Errorf("note = %q with the copy %q, want the head and the copy that exists", got, outcome.Copy)
	}
}

func TestResolve_WritesTheHeadOverAnotherKindOfFileAndMakesNoCopy(t *testing.T) {
	s := newSetup(t)
	s.root.File(pdf, "edited here")
	in := s.input(t, pdf, "edited there", new(int))

	outcome, err := s.res.Resolve(t.Context(), in)

	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.text(pdf); got != "edited there" {
		t.Errorf("file = %q, want the head", got)
	}
	if outcome.Copy != "" || !slices.Equal(outcome.Applied, []string{pdf}) {
		t.Errorf("outcome = %+v, want only the file written and no copy", outcome)
	}
	if _, ok := s.text("Papers/Report (conflict 2026-11-23).md"); ok {
		t.Error("a copy of a file that is not a note was made")
	}
}

func TestResolve_ReplacesAFileWithTheBaselineOfTheLocalStat(t *testing.T) {
	s := newSetup(t)
	s.root.File(pdf, "edited here")
	in := s.input(t, pdf, "edited there", new(int))
	var baseline domain.FileStat
	in.Head.Replace = func(_ context.Context, got domain.FileStat) error {
		baseline = got
		return nil
	}

	if _, err := s.res.Resolve(t.Context(), in); err != nil {
		t.Fatal(err)
	}

	if baseline != in.Local.Stat {
		t.Errorf("baseline = %+v, want the stat of the local file %+v", baseline, in.Local.Stat)
	}
}

func TestResolve_LogsOneLineWithThePathAndBothHashesForAFileThatIsNotANote(t *testing.T) {
	s := newSetup(t)
	s.root.File(pdf, "edited here")
	in := s.input(t, pdf, "edited there", new(int))

	if _, err := s.res.Resolve(t.Context(), in); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(s.logs.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], pdf) || !strings.Contains(lines[0], in.Local.Hash) || !strings.Contains(lines[0], in.Head.Hash) {
		t.Errorf("log = %q, want one line with the path, the local hash and the head hash", lines)
	}
}

func TestResolve_ReturnsTheErrorOfTheHeadAndKeepsTheLocalFileOfAnotherKind(t *testing.T) {
	s := newSetup(t)
	s.root.File(pdf, "edited here")
	in := s.input(t, pdf, "edited there", new(int))
	refused := errors.New("the host refuses the file")
	in.Head.Replace = func(context.Context, domain.FileStat) error { return refused }

	_, err := s.res.Resolve(t.Context(), in)

	if !errors.Is(err, refused) {
		t.Errorf("error = %v, want the error of the head", err)
	}
	if got, _ := s.text(pdf); got != "edited here" {
		t.Errorf("file = %q, want the local bytes", got)
	}
}

func TestResolve_NamesTheCopyOfANoteInAFolderInItsFolder(t *testing.T) {
	s := newSetup(t)
	s.root.File("Daily/Monday.md", "tea")

	outcome, err := s.res.Resolve(t.Context(), s.input(t, "Daily/Monday.md", "rice", new(int)))

	if err != nil {
		t.Fatal(err)
	}
	if want := "Daily/Monday (conflict 2026-11-23).md"; outcome.Copy != want {
		t.Errorf("copy = %q, want %q", outcome.Copy, want)
	}
}

func TestResolve_UsesTheDayOfTheInput(t *testing.T) {
	s := newSetup(t)
	s.root.File(list, "eggs")
	in := s.input(t, list, "bread", new(int))
	in.Day = time.Date(2027, time.January, 5, 8, 0, 0, 0, time.UTC)

	outcome, err := s.res.Resolve(t.Context(), in)

	if err != nil {
		t.Fatal(err)
	}
	if want := "Shopping list (conflict 2027-01-05).md"; outcome.Copy != want {
		t.Errorf("copy = %q, want %q", outcome.Copy, want)
	}
}
