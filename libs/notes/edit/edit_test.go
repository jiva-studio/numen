package edit_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// files are the files of a vault held in memory. A write moves the modification time on, and one held
type files struct {
	content map[string]string
	kinds   map[string]domain.SourceKind
	clock   time.Time
	stats   map[string]domain.FileStat
	locks   int
}

func newFiles(content map[string]string) *files {
	f := &files{
		content: map[string]string{}, kinds: map[string]domain.SourceKind{}, stats: map[string]domain.FileStat{},
		clock: time.Unix(1_700_000_000, 0),
	}
	for path, body := range content {
		f.put(path, body)
	}
	return f
}

func (f *files) put(path, body string) domain.FileStat {
	f.clock = f.clock.Add(time.Second)
	f.content[path] = body
	f.stats[path] = domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: f.clock}
	return f.stats[path]
}

func (f *files) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	fp, there := f.stats[path]
	if !there {
		return domain.FileStat{}, fs.ErrNotExist
	}
	if kind, set := f.kinds[path]; set {
		fp.Kind = kind
	}
	return fp, nil
}

func (f *files) Read(_ context.Context, path string) ([]byte, error) {
	body, there := f.content[path]
	if !there {
		return nil, fs.ErrNotExist
	}
	return []byte(body), nil
}

func (f *files) Write(_ context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	if !stat.IsZero() && !f.stats[path].IsUnchanged(stat) {
		return domain.FileStat{}, edit.ErrStale
	}
	return f.put(path, string(content)), nil
}

func (f *files) Lock(context.Context) (func(), error) {
	f.locks++
	return func() { f.locks-- }, nil
}

var vault = domain.Vault{ID: "v1", Name: "Physics", Path: "/vaults/Physics"}

func editing(f *files) edit.Edit {
	return edit.Edit{
		Open: func(domain.Vault) (edit.Files, error) { return f, nil },
		Now:  func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
}

func TestANoteIsReadAsItsProse(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "---\ntitle: A\n---\n\nhello\r\nworld\n"})

	got, err := edit.Read(t.Context(), f, "A.md")

	if err != nil || got.Outcome != edit.Ok || got.Body != "\nhello\nworld\n" || got.FileStat.Path != "A.md" {
		t.Errorf("got %+v (%v)", got, err)
	}
}

func TestAReadReturnsWhatIsWrongWithTheNoteAndNotAnError(t *testing.T) {
	big := newFiles(map[string]string{"Big.md": strings.Repeat("x", edit.MaxBytes+1)})
	binary := newFiles(map[string]string{"Bin.md": "\xff\xfe"})
	broken := newFiles(map[string]string{"Broken.md": "---\n: [\n---\nbody"})
	book := newFiles(map[string]string{"B.epub": "x"})
	book.kinds["B.epub"] = domain.KindFile
	cases := []struct {
		name  string
		files *files
		path  string
		want  edit.ReadOutcome
	}{
		{"nothing there", newFiles(nil), "None.md", edit.Missing},
		{"too large", big, "Big.md", edit.TooLarge},
		{"not text", binary, "Bin.md", edit.NotText},
		{"frontmatter that is not YAML", broken, "Broken.md", edit.Unreadable},
		{"not a note", book, "B.epub", edit.NotANote},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			got, err := edit.Read(t.Context(), one.files, one.path)

			if err != nil || got.Outcome != one.want || got.Body != "" {
				t.Errorf("got %+v (%v)", got, err)
			}
		})
	}
}

func TestApply_StampsIDAndLocksWrite(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, _ := f.StatFile(t.Context(), "A.md")

	res, err := editing(f).Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if err != nil || res.IsUnchanged(read) {
		t.Fatalf("got %+v (%v)", res, err)
	}
	if got := f.content["A.md"]; !strings.Contains(got, "id:") || !strings.HasSuffix(got, "# B\n") {
		t.Errorf("the note is now %q", got)
	}
	if f.locks != 0 {
		t.Errorf("%d locks were left held", f.locks)
	}
}

func TestAnEditOverANoteThatMovedIsRefused(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, _ := f.StatFile(t.Context(), "A.md")
	f.put("A.md", "# A, edited by somebody\n")
	e := editing(f)
	e.FileStat = read

	_, err := e.Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrStale) || f.content["A.md"] != "# A, edited by somebody\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestABaselineOfTheSameProseLandsWhateverTheFileReturns(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, _ := f.StatFile(t.Context(), "A.md")
	f.put("A.md", "# A\n")
	e := editing(f)
	e.ShouldOverwrite = true
	e.Baseline = &edit.Baseline{Prose: "# A\n", FileStat: read}

	_, err := e.Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if err != nil || f.content["A.md"] != "# B\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestABaselineOfOtherProseIsStale(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A, changed\n"})
	e := editing(f)
	e.ShouldOverwrite = true
	e.Baseline = &edit.Baseline{Prose: "# A\n", FileStat: domain.FileStat{Path: "A.md", Size: 3}}

	_, err := e.Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("got %v", err)
	}
}

func TestApply_OverwriteCreatesMissingNoteElseErrors(t *testing.T) {
	f := newFiles(nil)
	e := editing(f)
	e.ShouldOverwrite = true
	if _, err := e.Apply(t.Context(), vault, "New.md", func(doc *markdown.Document) error { return doc.SetBody("# New\n") }); err != nil {
		t.Fatal(err)
	}

	_, err := editing(f).Apply(t.Context(), vault, "Other.md", func(doc *markdown.Document) error { return doc.SetBody("x") })

	if f.content["New.md"] != "# New\n" || !errors.Is(err, edit.ErrNoNote) {
		t.Errorf("the new note is %q, and editing a missing one gave %v", f.content["New.md"], err)
	}
}

func TestEdit_EnforcesSizeLimit(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	e := editing(f)
	e.Bound = 10

	_, err := e.Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody(strings.Repeat("x", 50)) })

	if !errors.Is(err, edit.ErrTooLarge) || f.content["A.md"] != "# A\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestAWriteThatLandedIsReportedAlongsideALevellingThatFailed(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	e := editing(f)
	e.Index = func(context.Context, domain.Vault, []string) error { return errors.New("index is busy") }

	res, err := e.Apply(t.Context(), vault, "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrUnlevelled) || res.Size == 0 || !strings.HasSuffix(f.content["A.md"], "# B\n") {
		t.Errorf("got %+v (%v)", res, err)
	}
}

func TestANoteIsPointedAtAnotherBelowAType(t *testing.T) {
	f := newFiles(map[string]string{"Sample.md": "---\ntype: sample\n---\n\n# Sample\n"})
	e := editing(f)

	_, err := e.PointAt(t.Context(), vault, "Sample.md", "preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow"}, domain.RoleRef)

	if err != nil || !strings.Contains(f.content["Sample.md"], "Slow") {
		t.Errorf("got %v and the note %q", err, f.content["Sample.md"])
	}
}

type names map[string][]string

func (n names) GetNamedPaths(_ context.Context, _ domain.VaultID, name string) ([]string, error) {
	return n[name], nil
}

func TestANoteIsAddressedByItsNameUnlessAnotherSharesIt(t *testing.T) {
	shared := names{"Slow": {"a/Slow.md", "b/Slow.md"}, "Fast": {"Fast.md"}}

	byName, _ := edit.GetAddress(t.Context(), shared, "v1", "Fast.md")
	byPath, _ := edit.GetAddress(t.Context(), shared, "v1", "a/Slow.md")
	_, unreachable := edit.GetAddress(t.Context(), shared, "v1", "a/C#.md")

	if byName.Value != "Fast" || byPath.Value != "a/Slow" || !errors.Is(unreachable, edit.ErrUnaddressable) {
		t.Errorf("got %v and %v, and %v", byName, byPath, unreachable)
	}
}
