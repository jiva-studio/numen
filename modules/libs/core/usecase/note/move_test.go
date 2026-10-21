package note_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

// isolate points everything a test could reach by way of the user's home at
// a folder of its own.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

// staleLinks answers as the index does, except where it is told to fail or
// to say nothing.
type staleLinks struct {
	port.LinkQueries
	backlinksErr error
	linksErr     error
	isLinksEmpty bool
	emptyFor     string
}

func (f staleLinks) GetBacklinks(ctx context.Context, id domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	if f.backlinksErr != nil {
		return nil, f.backlinksErr
	}
	return f.LinkQueries.GetBacklinks(ctx, id, to)
}

func (f staleLinks) GetLinks(ctx context.Context, id domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	if f.linksErr != nil {
		return nil, f.linksErr
	}
	if f.isLinksEmpty || from == f.emptyFor {
		return nil, nil
	}
	return f.LinkQueries.GetLinks(ctx, id, from)
}

// failingNames is a vault that cannot say which notes share a name.
type failingNames struct{ err error }

func (f failingNames) GetNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, f.err
}

// failingWriters writes as the filesystem does, except where it is told to
// fail: to be held, to be opened for the nth time, or to write a file.
type failingWriters struct {
	port.VaultWriters
	holdErr  error
	openErr  error
	failOpen int
	writeErr error
	opened   *int
}

func (f failingWriters) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	if f.holdErr != nil {
		return nil, f.holdErr
	}
	return f.VaultWriters.Lock(ctx, v)
}

func (f failingWriters) Open(v domain.Vault) (port.VaultWriter, error) {
	*f.opened++
	if f.openErr != nil && *f.opened == f.failOpen {
		return nil, f.openErr
	}
	w, err := f.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return failingWriter{VaultWriter: w, err: f.writeErr}, nil
}

type failingWriter struct {
	port.VaultWriter
	err error
}

func (f failingWriter) Write(ctx context.Context, path string, content []byte, fp domain.Fingerprint) (domain.Fingerprint, error) {
	if f.err != nil {
		return domain.Fingerprint{}, f.err
	}
	return f.VaultWriter.Write(ctx, path, content, fp)
}

// moveParts are the collaborators of a move a test replaces. What is left
// unset is what a real caller hands it.
type moveParts struct {
	links    port.LinkQueries
	names    note.NameQueries
	readers  port.VaultReaders
	writers  port.VaultWriters
	index    note.Levels
	hasIndex bool
}

func (c changing) moveOf(p moveParts) note.Move {
	if p.links == nil {
		p.links = c.db.Links()
	}
	if p.names == nil {
		p.names = c.db.Queries()
	}
	if p.readers == nil {
		p.readers = filesystem.VaultReaders{}
	}
	if p.writers == nil {
		p.writers = filesystem.VaultWriters{}
	}
	if !p.hasIndex {
		p.index = c.index
	}
	return note.NewMove(p.readers, p.writers, p.links, p.names, c.db.Sources(), p.index, time.Now)
}

const heatLinkedToEntropy = "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n"

func openLinked(t *testing.T) changing {
	t.Helper()
	isolate(t)
	return openChanging(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    heatLinkedToEntropy,
	})
}

// A move to the path a note is at is not a move. The answer says which note it
// was asked about.
func TestMovingANoteToItsOwnPathAnswersWithThatPath(t *testing.T) {
	c := openLinked(t)

	res, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "physics/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if res.From != "physics/Entropy.md" || res.To != "physics/Entropy.md" {
		t.Errorf("the answer names %q and %q", res.From, res.To)
	}
	if res.IsLanded {
		t.Error("nothing moved and the answer says the file landed")
	}
}

// A move that stops before the file moved says where it was asked to go, and
// that the file did not.
func TestAMoveThatStopsBeforeTheFileMovedSaysSo(t *testing.T) {
	boom := errors.New("the vault would not do it")
	for name, c := range map[string]struct {
		from  string
		parts func(c changing) moveParts
		want  error
	}{
		"the links pointing at it cannot be read": {
			from: "physics/Entropy.md",
			parts: func(c changing) moveParts {
				return moveParts{links: staleLinks{LinkQueries: c.db.Links(), backlinksErr: boom}}
			},
			want: boom,
		},
		"the vault cannot be opened for writing": {
			from: "physics/Entropy.md",
			parts: func(changing) moveParts {
				return moveParts{writers: failingWriters{
					VaultWriters: filesystem.VaultWriters{}, openErr: boom, failOpen: 1, opened: new(int),
				}}
			},
			want: boom,
		},
		"the note is not there": {
			from:  "physics/Missing.md",
			parts: func(changing) moveParts { return moveParts{} },
			want:  note.ErrNoNote,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c2 := openLinked(t)
			res, err := c2.moveOf(c.parts(c2)).Execute(t.Context(), c2.vault, c.from, "archive/Thermodynamics.md")
			if !errors.Is(err, c.want) {
				t.Fatalf("want %v, got %v", c.want, err)
			}
			if res.From != c.from || res.To != "archive/Thermodynamics.md" {
				t.Errorf("the answer names %q and %q", res.From, res.To)
			}
			if res.IsLanded {
				t.Error("the file did not move and the answer says it landed")
			}
			if !c2.exists("physics/Entropy.md") {
				t.Error("the note left its place")
			}
		})
	}
}

// Whoever is drawing the note is told where it went.
func TestAMoveTellsWhoIsDrawingTheNoteWhereItWent(t *testing.T) {
	c := openLinked(t)
	var notified []domain.Move
	moving := c.move()
	moving.Drawing = func(_ context.Context, went domain.Move) { notified = append(notified, went) }

	if _, err := moving.Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md"); err != nil {
		t.Fatal(err)
	}
	want := []domain.Move{{From: "physics/Entropy.md", To: "archive/Thermodynamics.md"}}
	if !slices.Equal(notified, want) {
		t.Errorf("told %+v", notified)
	}
}

// A note nothing points at has no link to name an address for, so a vault that
// cannot say which names are shared is not asked, and the index is not brought
// level with notes nothing was written into.
func TestAMoveNothingPointsAtAsksNothingAboutLinks(t *testing.T) {
	c := openLinked(t)
	var request [][]string
	parts := moveParts{
		names:    failingNames{err: errors.New("no names")},
		hasIndex: true,
		index: func(_ context.Context, _ domain.Vault, paths []string) error {
			request = append(request, paths)
			return errors.New("the index was asked")
		},
	}

	res, err := c.moveOf(parts).Execute(t.Context(), c.vault, "physics/Heat.md", "archive/Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsLanded || len(res.Repaired) != 0 || len(res.Dangling) != 0 {
		t.Errorf("got %+v", res)
	}
	if len(request) != 0 {
		t.Errorf("the index was brought level with %v", request)
	}
}

// The work after the file moved stops at the first thing that fails, and says
// so, and the file stays where it landed.
func TestAMoveThatFailsAfterTheFileLandedSaysWhatFailed(t *testing.T) {
	boom := errors.New("the vault would not do it")
	unterminated := "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n# Heat\n"
	for name, c := range map[string]struct {
		before     func(t *testing.T, c changing)
		parts      func(c changing) moveParts
		want       error
		isRepaired bool
	}{
		"the names cannot be asked": {
			parts: func(changing) moveParts { return moveParts{names: failingNames{err: boom}} },
			want:  boom,
		},
		"the links of the note holding the link cannot be read": {
			parts: func(c changing) moveParts {
				return moveParts{links: staleLinks{LinkQueries: c.db.Links(), linksErr: boom}}
			},
			want: boom,
		},
		"the vault cannot be held": {
			parts: func(changing) moveParts {
				return moveParts{writers: failingWriters{
					VaultWriters: filesystem.VaultWriters{}, holdErr: boom, opened: new(int),
				}}
			},
			want: boom,
		},
		"the vault cannot be read": {
			parts: func(changing) moveParts { return moveParts{readers: brokenReaders{open: boom}} },
			want:  boom,
		},
		"the vault cannot be opened again for the repair": {
			parts: func(changing) moveParts {
				return moveParts{writers: failingWriters{
					VaultWriters: filesystem.VaultWriters{}, openErr: boom, failOpen: 2, opened: new(int),
				}}
			},
			want: boom,
		},
		"the repaired note cannot be written": {
			parts: func(changing) moveParts {
				return moveParts{writers: failingWriters{
					VaultWriters: filesystem.VaultWriters{}, writeErr: boom, opened: new(int),
				}}
			},
			want: boom,
		},
		"the note holding the link opens a frontmatter block it never closes": {
			before: func(t *testing.T, c changing) { c.overwrite(t, "physics/Heat.md", unterminated) },
			parts:  func(changing) moveParts { return moveParts{} },
			want:   markdown.ErrUnterminated,
		},
		"the index cannot be brought level": {
			parts: func(changing) moveParts {
				return moveParts{hasIndex: true, index: func(context.Context, domain.Vault, []string) error { return boom }}
			},
			want:       note.ErrUnlevelled,
			isRepaired: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c2 := openLinked(t)
			if c.before != nil {
				c.before(t, c2)
			}
			before := c2.read(t, "physics/Heat.md")

			res, err := c2.moveOf(c.parts(c2)).Execute(t.Context(), c2.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
			if !errors.Is(err, c.want) {
				t.Fatalf("want %v, got %v", c.want, err)
			}
			if !res.IsLanded {
				t.Error("the file landed and the answer says it did not")
			}
			if !c2.exists("archive/Thermodynamics.md") {
				t.Error("the file is not where it was sent")
			}
			if c.isRepaired {
				return
			}
			if got := c2.read(t, "physics/Heat.md"); got != before {
				t.Errorf("a note that could not be repaired was written:\n%s", got)
			}
		})
	}
}

// A link the note no longer holds is not repaired and not reported.
func TestALinkTheNoteNoLongerHoldsIsLeftOut(t *testing.T) {
	c := openLinked(t)
	parts := moveParts{links: staleLinks{LinkQueries: c.db.Links(), isLinksEmpty: true}}

	res, err := c.moveOf(parts).Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) != 0 || len(res.Dangling) != 0 {
		t.Errorf("got %+v", res)
	}
	if got := c.read(t, "physics/Heat.md"); got != heatLinkedToEntropy {
		t.Errorf("the note was written:\n%s", got)
	}
}

// A note that went since the links were read took its link with it.
func TestANoteThatWentSinceTheLinksWereReadIsLeftOut(t *testing.T) {
	c := openLinked(t)
	if err := os.Remove(filepath.Join(c.vault.Path, "physics", "Heat.md")); err != nil {
		t.Fatal(err)
	}

	res, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) != 0 || len(res.Dangling) != 0 {
		t.Errorf("got %+v", res)
	}
}

// A note that no longer says the address the index read from it has nothing to
// repair, and is not written.
func TestANoteThatNoLongerSaysTheLinkIsNotWritten(t *testing.T) {
	c := openLinked(t)
	c.overwrite(t, "physics/Heat.md", "# Heat\n")

	res, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) != 0 || len(res.Dangling) != 0 {
		t.Errorf("got %+v", res)
	}
	if got := c.read(t, "physics/Heat.md"); got != "# Heat\n" {
		t.Errorf("the note was written:\n%s", got)
	}
}

// A link in the block and a link in the prose are both repaired.
func TestALinkInTheBlockAndOneInTheProseAreBothRepaired(t *testing.T) {
	isolate(t)
	c := openChanging(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md": "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n" +
			"# Heat\nSee [[physics/Entropy.md]].\n",
	})

	res, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Repaired, []string{"physics/Heat.md"}) {
		t.Fatalf("the notes written again are %v", res.Repaired)
	}
	if body := c.read(t, "physics/Heat.md"); strings.Contains(body, "physics/Entropy.md") {
		t.Errorf("a path is still written:\n%s", body)
	}
}

// An installation with no index to bring level repairs all the same.
func TestARepairNeedsNoIndexToBringLevel(t *testing.T) {
	c := openLinked(t)

	res, err := c.moveOf(moveParts{hasIndex: true}).Execute(
		t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Repaired, []string{"physics/Heat.md"}) {
		t.Errorf("the notes written again are %v", res.Repaired)
	}
}

// A note whose name carries a character a link is read up to is reached by no
// link. Moving it is not refused, and nothing is written for it.
func TestANoteNoLinkReachesCanStillBeMoved(t *testing.T) {
	c := openLinked(t)

	res, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Hot#Air.md")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsLanded {
		t.Error("the file did not land")
	}
	if len(res.Repaired) != 0 {
		t.Errorf("a link was written for a name no link reaches: %v", res.Repaired)
	}
	if got := c.read(t, "physics/Heat.md"); got != heatLinkedToEntropy {
		t.Errorf("the note was written:\n%s", got)
	}
}

// A link that is gone from one note does not stop the notes after it from
// being repaired.
func TestALinkTheNoteNoLongerHoldsDoesNotStopTheNextRepair(t *testing.T) {
	isolate(t)
	c := openChanging(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Cold.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Cold\n",
		"physics/Heat.md":    heatLinkedToEntropy,
	})
	parts := moveParts{links: staleLinks{LinkQueries: c.db.Links(), emptyFor: "physics/Cold.md"}}

	res, err := c.moveOf(parts).Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Repaired, []string{"physics/Heat.md"}) {
		t.Errorf("the notes written again are %v", res.Repaired)
	}
}

// A note that cannot be read is named in the error, and the reason stays in it.
func TestAMoveNamesTheNoteItCouldNotRead(t *testing.T) {
	c := openLinked(t)
	boom := errors.New("the disk would not answer")

	_, err := c.moveOf(moveParts{readers: brokenReaders{read: boom}}).Execute(
		t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "physics/Heat.md") {
		t.Errorf("the note is not named: %v", err)
	}
}

// A note whose frontmatter is written on one line cannot be repaired a key at a
// time, and is not written.
func TestANoteWithInlineFrontmatterIsReportedAndNotWritten(t *testing.T) {
	c := openLinked(t)
	inline := "---\n{title: Heat, links: []}\n---\n# Heat\n"
	c.overwrite(t, "physics/Heat.md", inline)

	_, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if !errors.Is(err, markdown.ErrInline) {
		t.Fatalf("got %v", err)
	}
	if got := c.read(t, "physics/Heat.md"); got != inline {
		t.Errorf("the note was written:\n%s", got)
	}
}

// The collaborators of a move are replaced one at a time, and each is given
// back as it was set.
func TestAMoveHandsBackTheCollaboratorsItWasGiven(t *testing.T) {
	c := openLinked(t)
	readers, writers := brokenReaders{}, &failingWriters{VaultWriters: filesystem.VaultWriters{}, opened: new(int)}
	sources := sulking{SourceRepository: c.db.Sources()}

	moving := c.move().WithReaders(readers).WithWriters(writers).WithSources(sources)
	if moving.Readers() != port.VaultReaders(readers) {
		t.Error("the readers are not the ones set")
	}
	if moving.Writers() != port.VaultWriters(writers) {
		t.Error("the writers are not the ones set")
	}
	if moving.Sources() != port.SourceRepository(sources) {
		t.Error("the sources are not the ones set")
	}
}

func TestAFolderKeepsTheNameOfTheNoteFiledUnderIt(t *testing.T) {
	if got := note.GetPathUnder("archive", "physics/Entropy.md"); got != "archive/Entropy.md" {
		t.Errorf("got %q", got)
	}
}
