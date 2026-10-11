package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

var errRefused = errors.New("refused on purpose")

// notebook is the vault every command here is asked about.
var notebook = domain.Vault{ID: "vlt-01", Name: "notebook", Path: "/nowhere/notebook"}

// runWith runs the terminal on the values it is handed.
func runWith(given cli.Deps, args ...string) (string, error) {
	var out bytes.Buffer
	err := cli.Run(context.Background(), &out, io.Discard, args, func(cli.Locations) cli.Deps { return given })
	return out.String(), err
}

func listing(held domain.Vault) func() (cli.Vaults, error) {
	return func() (cli.Vaults, error) { return cli.Vaults{Registry: oneVault{held}}, nil }
}

// unwilling answers every question about links with an error.
type unwilling struct{}

func (unwilling) Links(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errRefused
}

func (unwilling) Backlinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errRefused
}

func (unwilling) Resolve(context.Context, domain.VaultID, string, []string) (map[string]domain.ResolvedLink, error) {
	return nil, errRefused
}

func linksOf(links, backlinks []domain.ResolvedLink) cli.Deps {
	return cli.Deps{
		Vaults: listing(notebook),
		Links: func(context.Context) (cli.Links, error) {
			return cli.Links{Show: note.NewShowLinks(joined{links: links, backlinks: backlinks})}, nil
		},
	}
}

func link(role domain.LinkRole, scheme, value, to string) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: scheme, Value: value}, Role: role},
		From: "Bees.md",
		To:   to,
	}
}

func TestLinksReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	for name, given := range map[string]cli.Deps{
		"the list of vaults cannot be opened": {
			Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused },
		},
		"the links cannot be opened": {
			Vaults: listing(notebook),
			Links:  func(context.Context) (cli.Links, error) { return cli.Links{}, errRefused },
		},
		"the index cannot say what a note points at": {
			Vaults: listing(notebook),
			Links: func(context.Context) (cli.Links, error) {
				return cli.Links{Show: note.NewShowLinks(unwilling{})}, nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, err := runWith(given, "links", "notebook", "Bees.md")
			if !errors.Is(err, errRefused) {
				t.Errorf("links answered %v", err)
			}
			if out != "" {
				t.Errorf("links said something before it failed:\n%s", out)
			}
		})
	}
}

func TestLinksWithoutAVaultAndANoteSaysHowToAsk(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"links"}, {"links", "notebook"}, {"links", "notebook", "a.md", "b.md"}} {
		_, err := runWith(linksOf(nil, nil), args...)
		if err == nil || !strings.Contains(err.Error(), "usage: numen-cli links <vault> <note>") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}

func TestLinksSaysWhichDirectionsAreEmpty(t *testing.T) {
	t.Parallel()
	pointing := []domain.ResolvedLink{link(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")}
	pointed := []domain.ResolvedLink{link(domain.RoleRef, domain.SchemeName, "Bees", "Bees.md")}
	pointed[0].From = "Honey.md"

	for name, c := range map[string]struct {
		links, backlinks []domain.ResolvedLink
		want             string
	}{
		"neither":   {nil, nil, "no links either way\n"},
		"out only":  {pointing, nil, "points at:\n  child      Hives.md\n"},
		"in only":   {nil, pointed, "pointed at by:\n  ref        Honey.md\n"},
		"both ways": {pointing, pointed, "points at:\n  child      Hives.md\npointed at by:\n  ref        Honey.md\n"},
	} {
		out, err := runWith(linksOf(c.links, c.backlinks), "links", "notebook", "Bees.md")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if out != c.want {
			t.Errorf("%s: links said\n%q\nwant\n%q", name, out, c.want)
		}
	}
}

func TestLinksDescribesWhereEachLinkGoes(t *testing.T) {
	t.Parallel()
	ambiguous := link(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	ambiguous.IsAmbiguous = true
	crossed := link(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	crossed.ToVault = "vlt-02"
	home := link(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	home.ToVault = notebook.ID
	typed := link(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	typed.Type = "kin"
	reasoned := link(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	reasoned.Why = "they share a queen"
	both := link(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	both.Type, both.Why = "kin", "they share a queen"

	for name, c := range map[string]struct {
		link domain.ResolvedLink
		want string
	}{
		"a name nothing answers to": {
			link(domain.RoleRef, domain.SchemeName, "Ghost", ""),
			"ref        Ghost  (nothing by that name)",
		},
		"a note no connected vault holds": {
			link(domain.RoleRef, domain.SchemeNote, "abc", ""),
			"ref        note://abc  (no connected vault holds this note)",
		},
		"an address of another kind": {
			link(domain.RoleRef, domain.SchemeAsset, "bee.png", ""),
			"ref        asset://bee.png",
		},
		"a name that resolved":               {link(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md"), "child      Hives.md"},
		"a note address that resolved":       {link(domain.RoleChild, domain.SchemeNote, "abc", "Hives.md"), "child      Hives.md"},
		"an asset address that resolved":     {link(domain.RoleChild, domain.SchemeAsset, "bee.png", "bee.png"), "child      bee.png"},
		"a name several notes answer to":     {ambiguous, "child      Hives.md  (several notes answer to that name)"},
		"a note in another vault":            {crossed, "child      Hives.md  (in another vault: vlt-02)"},
		"a note in the vault it was written": {home, "child      Hives.md"},
		"a type":                             {typed, "ref        Hives.md  [kin]"},
		"a reason":                           {reasoned, "ref        Hives.md  — they share a queen"},
		"a type and a reason":                {both, "ref        Hives.md  [kin]  — they share a queen"},
	} {
		out, err := runWith(linksOf([]domain.ResolvedLink{c.link}, nil), "links", "notebook", "Bees.md")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if want := "points at:\n  " + c.want + "\n"; out != want {
			t.Errorf("%s: links said\n%q\nwant\n%q", name, out, want)
		}
	}
}

// problemsOf answers the two stored questions with what the test wrote down.
type problemsOf struct{ parse, unreadable []domain.VaultProblem }

func (p problemsOf) GetParseProblems(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return p.parse, nil
}

func (p problemsOf) GetUnreadable(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return p.unreadable, nil
}

func (problemsOf) Ambiguous(context.Context, domain.VaultID) ([]domain.AmbiguousLink, error) {
	return nil, nil
}

func (problemsOf) GetDanglingLinks(context.Context, domain.VaultID) ([]domain.ResolvedLink, error) {
	return nil, nil
}

func problemsDeps(queries port.ProblemQueries) cli.Deps {
	return cli.Deps{
		Vaults: listing(notebook),
		Problems: func(context.Context) (cli.Problems, error) {
			return cli.Problems{Problems: queries}, nil
		},
	}
}

func TestProblemsNamesTheChecksItRuns(t *testing.T) {
	t.Parallel()
	given := problemsDeps(problemsOf{
		parse:      []domain.VaultProblem{{Path: "a.md", Detail: "no role"}},
		unreadable: []domain.VaultProblem{{Path: "b.md", Detail: "bad yaml"}},
	})

	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"every loud check": {
			[]string{"problems", "notebook"},
			"a.md\n  parse: no role\nb.md\n  frontmatter: the frontmatter is not YAML, so nothing may be written here: bad yaml\n",
		},
		"the first one named": {
			[]string{"problems", "notebook", "parse"},
			"a.md\n  parse: no role\n",
		},
		"the second one named": {
			[]string{"problems", "notebook", "frontmatter"},
			"b.md\n  frontmatter: the frontmatter is not YAML, so nothing may be written here: bad yaml\n",
		},
		"two named, the later first": {
			[]string{"problems", "notebook", "ambiguous", "frontmatter"},
			"b.md\n  frontmatter: the frontmatter is not YAML, so nothing may be written here: bad yaml\n",
		},
		"two named, the later one the only that finds anything": {
			[]string{"problems", "notebook", "dangling", "parse"},
			"a.md\n  parse: no role\n",
		},
		"one that finds nothing": {
			[]string{"problems", "notebook", "dangling"},
			"nothing to report\n",
		},
	} {
		out, err := runWith(given, c.args...)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if out != c.want {
			t.Errorf("%s: problems said\n%q\nwant\n%q", name, out, c.want)
		}
	}
}

func TestProblemsNeedsAVault(t *testing.T) {
	t.Parallel()
	_, err := runWith(problemsDeps(problemsOf{}), "problems")
	if err == nil || !strings.Contains(err.Error(), "usage: numen-cli problems <vault>") {
		t.Errorf("problems with no vault answered %v", err)
	}
}

func TestProblemsReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		given cli.Deps
		args  []string
	}{
		"the list of vaults cannot be opened": {
			cli.Deps{Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused }},
			[]string{"problems", "notebook"},
		},
		"the problems cannot be opened": {
			cli.Deps{
				Vaults:   listing(notebook),
				Problems: func(context.Context) (cli.Problems, error) { return cli.Problems{}, errRefused },
			},
			[]string{"problems", "notebook"},
		},
		"a check is not one there is": {
			problemsDeps(problemsOf{}),
			[]string{"problems", "notebook", "nonsense"},
		},
	} {
		out, err := runWith(c.given, c.args...)
		if err == nil {
			t.Errorf("%s: problems succeeded", name)
		}
		if out != "" {
			t.Errorf("%s: problems said something before it failed:\n%s", name, out)
		}
	}
	if _, err := runWith(cli.Deps{Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused }},
		"problems", "notebook"); !errors.Is(err, errRefused) {
		t.Errorf("the refusal was lost: %v", err)
	}
}

func TestACommandGivesBackWhatItOpened(t *testing.T) {
	t.Parallel()
	closed := 0
	given := linksOf(nil, nil)
	given.Links = func(context.Context) (cli.Links, error) {
		return cli.Links{
			Show:  note.NewShowLinks(joined{}),
			Close: func() error { closed++; return nil },
		}, nil
	}
	if _, err := runWith(given, "links", "notebook", "Bees.md"); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Errorf("what links opened was closed %d times", closed)
	}
}

// A vault the list cannot find is reported with the list's own reason intact.
func TestAVaultTheListCannotReadKeepsTheReason(t *testing.T) {
	t.Parallel()
	given := cli.Deps{Vaults: func() (cli.Vaults, error) {
		return cli.Vaults{Registry: brokenList{}}, nil
	}}
	_, err := runWith(given, "links", "notebook", "Bees.md")
	if !errors.Is(err, errRefused) {
		t.Errorf("the list's reason was lost: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "numen-cli vault add notebook") {
		t.Errorf("the answer does not say what to do: %v", err)
	}
}

type brokenList struct{ oneVault }

func (brokenList) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, errRefused }

func TestAMistypedFlagIsAnErrorAndNothingElse(t *testing.T) {
	// Not parallel: the standard flag package reports on the process's stderr.
	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	was := os.Stderr
	os.Stderr = file
	t.Cleanup(func() { os.Stderr = was; file.Close() })

	opened := false
	var out bytes.Buffer
	runErr := cli.Run(context.Background(), &out, io.Discard, []string{"--no-such-flag", "help"},
		func(cli.Locations) cli.Deps { opened = true; return cli.Deps{} })

	if runErr == nil {
		t.Error("a flag nobody knows was accepted")
	}
	if opened {
		t.Error("an installation was opened for a command line that did not parse")
	}
	if said, _ := os.ReadFile(file.Name()); len(said) != 0 {
		t.Errorf("the flag package wrote to the terminal:\n%s", said)
	}
}

func TestEveryLocationIsHandedToTheOpener(t *testing.T) {
	t.Parallel()
	var got cli.Locations
	var out bytes.Buffer
	err := cli.Run(context.Background(), &out, io.Discard,
		[]string{"--index", "i.db", "--registry", "r.json", "--service-dir", ".mine", "help"},
		func(where cli.Locations) cli.Deps { got = where; return cli.Deps{} })
	if err != nil {
		t.Fatal(err)
	}
	if want := (cli.Locations{Index: "i.db", Registry: "r.json", ServiceDir: ".mine"}); got != want {
		t.Errorf("the opener was handed %+v", got)
	}
}

func TestAFlagMistypedAfterTheCommandIsAnError(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"vault", "add", "--no-such-flag"},
		{"vault", "add", "somewhere", "--no-such-flag"},
		{"vault", "erase", "--no-such-flag"},
		{"vault", "erase", "notebook", "--no-such-flag"},
	} {
		if _, err := runWith(cli.Deps{Vaults: listing(notebook)}, args...); err == nil {
			t.Errorf("%v was accepted", args)
		}
	}
}

func TestDescribeSources(t *testing.T) {
	t.Parallel()
	base := source.ExtractResult{Seen: 5, Extracted: 3, Unchanged: 2, Chunks: 40}
	for name, c := range map[string]struct {
		unreadable, vanished int
		want                 string
	}{
		"all read":        {0, 0, "5 sources: 3 read, 2 unchanged, 40 chunks"},
		"one unreadable":  {1, 0, "5 sources: 3 read, 2 unchanged, 40 chunks, 1 could not be read"},
		"one vanished":    {0, 1, "5 sources: 3 read, 2 unchanged, 40 chunks, 1 gone before they could be read"},
		"several of each": {2, 3, "5 sources: 3 read, 2 unchanged, 40 chunks, 2 could not be read, 3 gone before they could be read"},
	} {
		r := base
		r.Unreadable, r.Vanished = c.unreadable, c.vanished
		if got := cli.DescribeSources(r); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

// pointing answers every text with one direction.
type pointing struct{}

const width = 1024

func (pointing) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
}

func (pointing) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, width)
		out[i][0] = 1
	}
	return out, nil
}

func (pointing) Close() error { return nil }

// scanning is a scan of a vault on a real index, with what the test chooses to
// put in place of an installation's model.
type scanning struct {
	embedder   port.Embedder
	unembedded error
	summary    error
	rebuilt    bool
}

func (s *scanning) deps(t *testing.T, held domain.Vault) cli.Deps {
	t.Helper()
	indexPath := filepath.Join(t.TempDir(), "index.db")
	return cli.Deps{
		Vaults: listing(held),
		Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
			s.rebuilt = rebuild
			db, err := index.Open(ctx, indexPath)
			if err != nil {
				return cli.Scan{}, err
			}
			options := filesystem.Options{}
			readers := filesystem.VaultReaders{Options: options}
			store, err := getDerived(options).Open(v)
			if err != nil {
				_ = db.Close()
				return cli.Scan{}, err
			}
			notes := vaults.NewScan(readers, db.Vaults(),
				db.Notes().Cut(chunking.Sizes{}, chunking.Legibility{}),
				db.NoteQueries(), db.Maintenance())
			books := source.NewExtract(readers, db.Sources(), db.Sources())
			books.Derived, books.Documents = store, pdf.Documents{}
			vectors := source.NewEmbed(readers, db.Sources(), db.Sources())
			if s.embedder != nil {
				if err := db.FitVectors(ctx, width, s.embedder.Model().Recipe()); err != nil {
					_ = db.Close()
					return cli.Scan{}, err
				}
				vectors.Embedder = s.embedder
			}
			return cli.Scan{
				Read: vaults.NewReadWholeVault(notes, books, vectors),
				Summary: func(ctx context.Context) (domain.VaultSummary, error) {
					if s.summary != nil {
						return domain.VaultSummary{}, s.summary
					}
					return db.NoteQueries().Summary(ctx, v.ID)
				},
				Unembedded: s.unembedded,
				Close:      db.Close,
			}, nil
		},
	}
}

func oneNote(t *testing.T) domain.Vault {
	t.Helper()
	return testsupport.NewVault(t, map[string]string{"Bees.md": "# Bees\n\nBees make honey.\n"})
}

func TestScanReportsTheNotesItIndexed(t *testing.T) {
	t.Parallel()
	held := oneNote(t)
	s := &scanning{unembedded: errors.New("no model here")}
	out, err := runWith(s.deps(t, held), "scan", held.Name)
	if err != nil {
		t.Fatal(err)
	}
	if want := "not embedding " + held.Name + ": no model here\n"; !strings.HasPrefix(out, want) {
		t.Errorf("scan did not say why no vectors are made:\n%q", out)
	}
	for _, want := range []string{"  1 indexed\r", "index now holds 1 notes"} {
		if !strings.Contains(out, want) {
			t.Errorf("scan did not say %q:\n%q", want, out)
		}
	}
	for _, not := range []string{"sources:", "embedded", "vectors made", "reading "} {
		if strings.Contains(out, not) {
			t.Errorf("scan of one note and no model said %q:\n%q", not, out)
		}
	}
	if s.rebuilt {
		t.Error("a scan with no flag rebuilt the index")
	}
}

func TestScanReportsTheVectorsItMade(t *testing.T) {
	t.Parallel()
	held := oneNote(t)
	s := &scanning{embedder: pointing{}}
	out, err := runWith(s.deps(t, held), "scan", held.Name)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  1 embedded\r", "1 vectors made\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("scan did not say %q:\n%q", want, out)
		}
	}
	if strings.Contains(out, "not embedding") {
		t.Errorf("scan said no vectors were made while it made them:\n%q", out)
	}
}

func TestScanReportsTheBooksItRead(t *testing.T) {
	t.Parallel()
	held := testsupport.NewVault(t, nil)
	testsupport.WriteBook(t, held.Path, "bee.epub")
	s := &scanning{}
	out, err := runWith(s.deps(t, held), "scan", held.Name)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  reading bee.epub\r", "1 sources: 1 read, 0 unchanged, "} {
		if !strings.Contains(out, want) {
			t.Errorf("scan did not say %q:\n%q", want, out)
		}
	}
	if strings.Contains(out, "reading \r") {
		t.Errorf("scan named a book when none was open:\n%q", out)
	}
}

func TestScanRebuildsTheIndexWhenAsked(t *testing.T) {
	t.Parallel()
	held := oneNote(t)
	s := &scanning{}
	for _, args := range [][]string{
		{"scan", held.Name, "--rebuild-index"},
		{"scan", "--rebuild-index", held.Name},
	} {
		s.rebuilt = false
		if _, err := runWith(s.deps(t, held), args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !s.rebuilt {
			t.Errorf("%v did not rebuild the index", args)
		}
	}
}

func TestScanReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	held := oneNote(t)

	opening := (&scanning{}).deps(t, held)
	opening.Scan = func(context.Context, domain.Vault, bool) (cli.Scan, error) { return cli.Scan{}, errRefused }
	if _, err := runWith(opening, "scan", held.Name); !errors.Is(err, errRefused) {
		t.Errorf("a scan that cannot open answered %v", err)
	}

	summing := &scanning{summary: errRefused}
	out, err := runWith(summing.deps(t, held), "scan", held.Name)
	if !errors.Is(err, errRefused) {
		t.Errorf("a scan that cannot count answered %v", err)
	}
	if strings.Contains(out, "index now holds") {
		t.Errorf("scan reported a count it did not have:\n%q", out)
	}

	gone := domain.Vault{ID: held.ID, Name: held.Name, Path: filepath.Join(t.TempDir(), "not-there")}
	if _, err := runWith((&scanning{}).deps(t, gone), "scan", gone.Name); err == nil {
		t.Error("a scan of a folder that is not there succeeded")
	}
}
