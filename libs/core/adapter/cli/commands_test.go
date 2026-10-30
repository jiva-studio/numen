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

	"github.com/jiva-studio/numen/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/adapter/index"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

var errRefused = errors.New("rejected on purpose")

// notebook is the vault every command here is asked about.
var notebook = domain.Vault{ID: "vlt-01", Name: "notebook", Path: "/nowhere/notebook"}

// runWithDeps runs the terminal on the values it is handed.
func runWithDeps(deps cli.Deps, args ...string) (string, error) {
	var out bytes.Buffer
	err := cli.Run(context.Background(), &out, io.Discard, args, func(cli.Locations) cli.Deps { return deps })
	return out.String(), err
}

func createVaultList(target domain.Vault) func() (cli.Vaults, error) {
	return func() (cli.Vaults, error) { return cli.Vaults{Registry: singleVaultList{target}}, nil }
}

// failingLinkQueries answers every question about links with an error.
type failingLinkQueries struct{}

func (failingLinkQueries) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errRefused
}

func (failingLinkQueries) GetBacklinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errRefused
}

func (failingLinkQueries) ResolveLinks(context.Context, domain.VaultID, string, []string) (map[string]domain.ResolvedLink, error) {
	return nil, errRefused
}

func createLinkDeps(links, backlinks []domain.ResolvedLink) cli.Deps {
	return cli.Deps{
		Vaults: createVaultList(notebook),
		Links: func(context.Context) (cli.Links, error) {
			return cli.Links{Show: note.NewShowLinks(fixedLinkQueries{links: links, backlinks: backlinks})}, nil
		},
	}
}

func newResolvedLink(role domain.LinkRole, scheme, value, to string) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: scheme, Value: value}, Role: role},
		From: "Bees.md",
		To:   to,
	}
}

func TestLinksReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	for name, provided := range map[string]cli.Deps{
		"the list of vaults cannot be opened": {
			Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused },
		},
		"the links cannot be opened": {
			Vaults: createVaultList(notebook),
			Links:  func(context.Context) (cli.Links, error) { return cli.Links{}, errRefused },
		},
		"the index cannot say what a note points at": {
			Vaults: createVaultList(notebook),
			Links: func(context.Context) (cli.Links, error) {
				return cli.Links{Show: note.NewShowLinks(failingLinkQueries{})}, nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, err := runWithDeps(provided, "links", "notebook", "Bees.md")
			if !errors.Is(err, errRefused) {
				t.Errorf("links answered %v", err)
			}
			if out != "" {
				t.Errorf("links said something before it failed:\n%s", out)
			}
		})
	}
}

func TestLinksWithoutAVaultAndANoteReturnsHowToAsk(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"links"}, {"links", "notebook"}, {"links", "notebook", "a.md", "b.md"}} {
		_, err := runWithDeps(createLinkDeps(nil, nil), args...)
		if err == nil || !strings.Contains(err.Error(), "usage: numen-cli links <vault> <note>") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}

func TestLinksReturnsWhichDirectionsAreEmpty(t *testing.T) {
	t.Parallel()
	links := []domain.ResolvedLink{newResolvedLink(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")}
	pointed := []domain.ResolvedLink{newResolvedLink(domain.RoleRef, domain.SchemeName, "Bees", "Bees.md")}
	pointed[0].From = "Honey.md"

	for name, c := range map[string]struct {
		links, backlinks []domain.ResolvedLink
		want             string
	}{
		"neither":   {nil, nil, "no links either way\n"},
		"out only":  {links, nil, "points at:\n  child      Hives.md\n"},
		"in only":   {nil, pointed, "pointed at by:\n  ref        Honey.md\n"},
		"both ways": {links, pointed, "points at:\n  child      Hives.md\npointed at by:\n  ref        Honey.md\n"},
	} {
		out, err := runWithDeps(createLinkDeps(c.links, c.backlinks), "links", "notebook", "Bees.md")
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
	ambiguous := newResolvedLink(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	ambiguous.IsAmbiguous = true
	crossed := newResolvedLink(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	crossed.ToVault = "vlt-02"
	home := newResolvedLink(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md")
	home.ToVault = notebook.ID
	refLink := newResolvedLink(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	refLink.Type = "kin"
	reasoned := newResolvedLink(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	reasoned.Why = "they share a queen"
	both := newResolvedLink(domain.RoleRef, domain.SchemeName, "Hives", "Hives.md")
	both.Type, both.Why = "kin", "they share a queen"

	for name, c := range map[string]struct {
		link domain.ResolvedLink
		want string
	}{
		"a name nothing answers to": {
			newResolvedLink(domain.RoleRef, domain.SchemeName, "Ghost", ""),
			"ref        Ghost  (nothing by that name)",
		},
		"a note no connected vault holds": {
			newResolvedLink(domain.RoleRef, domain.SchemeNote, "abc", ""),
			"ref        note://abc  (no connected vault holds this note)",
		},
		"an address of another kind": {
			newResolvedLink(domain.RoleRef, domain.SchemeAsset, "bee.png", ""),
			"ref        asset://bee.png",
		},
		"a name that resolved":               {newResolvedLink(domain.RoleChild, domain.SchemeName, "Hives", "Hives.md"), "child      Hives.md"},
		"a note address that resolved":       {newResolvedLink(domain.RoleChild, domain.SchemeNote, "abc", "Hives.md"), "child      Hives.md"},
		"an asset address that resolved":     {newResolvedLink(domain.RoleChild, domain.SchemeAsset, "bee.png", "bee.png"), "child      bee.png"},
		"a name several notes answer to":     {ambiguous, "child      Hives.md  (several notes answer to that name)"},
		"a note in another vault":            {crossed, "child      Hives.md  (in another vault: vlt-02)"},
		"a note in the vault it was written": {home, "child      Hives.md"},
		"a type":                             {refLink, "ref        Hives.md  [kin]"},
		"a reason":                           {reasoned, "ref        Hives.md  — they share a queen"},
		"a type and a reason":                {both, "ref        Hives.md  [kin]  — they share a queen"},
	} {
		out, err := runWithDeps(createLinkDeps([]domain.ResolvedLink{c.link}, nil), "links", "notebook", "Bees.md")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if want := "points at:\n  " + c.want + "\n"; out != want {
			t.Errorf("%s: links said\n%q\nwant\n%q", name, out, want)
		}
	}
}

// fixedProblemQueries answers the two stored questions with what the test wrote down.
type fixedProblemQueries struct{ parse, unreadable []domain.VaultProblem }

func (p fixedProblemQueries) GetParseProblems(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return p.parse, nil
}

func (p fixedProblemQueries) GetUnreadable(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return p.unreadable, nil
}

func (fixedProblemQueries) GetAmbiguousLinks(context.Context, domain.VaultID) ([]domain.AmbiguousLink, error) {
	return nil, nil
}

func (fixedProblemQueries) GetDanglingLinks(context.Context, domain.VaultID) ([]domain.ResolvedLink, error) {
	return nil, nil
}

func problemsDeps(queries problemport.ProblemQueries) cli.Deps {
	return cli.Deps{
		Vaults: createVaultList(notebook),
		Problems: func(context.Context) (cli.Problems, error) {
			return cli.Problems{Problems: queries}, nil
		},
	}
}

func TestProblemsNamesTheChecksItRuns(t *testing.T) {
	t.Parallel()
	provided := problemsDeps(fixedProblemQueries{
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
		out, err := runWithDeps(provided, c.args...)
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
	_, err := runWithDeps(problemsDeps(fixedProblemQueries{}), "problems")
	if err == nil || !strings.Contains(err.Error(), "usage: numen-cli problems <vault>") {
		t.Errorf("problems with no vault answered %v", err)
	}
}

func TestProblemsReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		input cli.Deps
		args  []string
	}{
		"the list of vaults cannot be opened": {
			cli.Deps{Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused }},
			[]string{"problems", "notebook"},
		},
		"the problems cannot be opened": {
			cli.Deps{
				Vaults:   createVaultList(notebook),
				Problems: func(context.Context) (cli.Problems, error) { return cli.Problems{}, errRefused },
			},
			[]string{"problems", "notebook"},
		},
		"a check is not one there is": {
			problemsDeps(fixedProblemQueries{}),
			[]string{"problems", "notebook", "nonsense"},
		},
	} {
		out, err := runWithDeps(c.input, c.args...)
		if err == nil {
			t.Errorf("%s: problems succeeded", name)
		}
		if out != "" {
			t.Errorf("%s: problems said something before it failed:\n%s", name, out)
		}
	}
	if _, err := runWithDeps(cli.Deps{Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused }},
		"problems", "notebook"); !errors.Is(err, errRefused) {
		t.Errorf("the rejection was lost: %v", err)
	}
}

func TestACommandProducesBackWhatItOpened(t *testing.T) {
	t.Parallel()
	closed := 0
	provided := createLinkDeps(nil, nil)
	provided.Links = func(context.Context) (cli.Links, error) {
		return cli.Links{
			Show:  note.NewShowLinks(fixedLinkQueries{}),
			Close: func() error { closed++; return nil },
		}, nil
	}
	if _, err := runWithDeps(provided, "links", "notebook", "Bees.md"); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Errorf("what links opened was closed %d times", closed)
	}
}

// A vault the list cannot find is reported with the list's own reason intact.
func TestAVaultTheListCannotReadKeepsTheReason(t *testing.T) {
	t.Parallel()
	provided := cli.Deps{Vaults: func() (cli.Vaults, error) {
		return cli.Vaults{Registry: brokenVaultList{}}, nil
	}}
	_, err := runWithDeps(provided, "links", "notebook", "Bees.md")
	if !errors.Is(err, errRefused) {
		t.Errorf("the list's reason was lost: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "numen-cli vault add notebook") {
		t.Errorf("the answer does not say what to do: %v", err)
	}
}

type brokenVaultList struct{ singleVaultList }

func (brokenVaultList) FindVault(string) (domain.Vault, bool, error) {
	return domain.Vault{}, false, errRefused
}

func TestAMistypedFlagIsAnErrorAndNothingElse(t *testing.T) {
	// Not parallel: the standard flag package reports on the process's stderr.
	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	was := os.Stderr
	os.Stderr = file
	t.Cleanup(func() { os.Stderr = was })
	testsupport.CloseAfter(t, file)

	wasOpened := false
	var out bytes.Buffer
	runErr := cli.Run(t.Context(), &out, io.Discard, []string{"--no-such-flag", "help"},
		func(cli.Locations) cli.Deps { wasOpened = true; return cli.Deps{} })

	if runErr == nil {
		t.Error("a flag nobody knows was accepted")
	}
	if wasOpened {
		t.Error("an installation was opened for a command line that did not parse")
	}
	response, readErr := os.ReadFile(file.Name())
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(response) != 0 {
		t.Errorf("the flag package wrote to the terminal:\n%s", response)
	}
}

func TestEveryLocationIsHandedToTheOpener(t *testing.T) {
	t.Parallel()
	var got cli.Locations
	var out bytes.Buffer
	err := cli.Run(t.Context(), &out, io.Discard,
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
		_, err := runWithDeps(cli.Deps{Vaults: createVaultList(notebook)}, args...)
		if err == nil || !strings.Contains(err.Error(), "no-such-flag") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}

func TestDescribeSources(t *testing.T) {
	t.Parallel()
	base := filetext.IndexResult{Seen: 5, Extracted: 3, Unchanged: 2, Chunks: 40}
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

// constantEmbedder answers every text with one direction.
type constantEmbedder struct{}

const width = 1024

func (constantEmbedder) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
}

func (constantEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, width)
		out[i][0] = 1
	}
	return out, nil
}

func (constantEmbedder) Close() error { return nil }

// scanFixture is a scan of a vault on a real index, with what the test chooses to
// put in place of an installation's model.
type scanFixture struct {
	embedder   embeddingport.Embedder
	unembedded error
	summary    error
	rebuilt    bool
}

func (s *scanFixture) deps(t *testing.T, target domain.Vault) cli.Deps {
	t.Helper()
	indexPath := filepath.Join(t.TempDir(), "index.db")
	return cli.Deps{
		Vaults: createVaultList(target),
		Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
			s.rebuilt = rebuild
			db, err := index.Open(ctx, indexPath)
			if err != nil {
				return cli.Scan{}, err
			}
			options := vaultconfig.Options{}
			readers := vaultreader.Readers{Options: options}
			notes := vaults.NewScan(readers, db.Vaults(),
				db.Notes(),
				db.NoteQueries(), db.Maintenance())
			books := filetext.NewIndex(readers, db.Sources(), db.Sources(), testsupport.Texts{Readers: readers})
			embed := vectors.NewEmbed(readers, db.Sources(), db.Sources())
			if s.embedder != nil {
				if err := db.FitVectors(ctx, width, s.embedder.GetModel().Recipe()); err != nil {
					testsupport.Close(t, db)
					return cli.Scan{}, err
				}
				embed.Embedder = s.embedder
			}
			return cli.Scan{
				Read: vaults.NewReadWholeVault(notes, books, embed),
				Summary: func(ctx context.Context) (domain.VaultSummary, error) {
					if s.summary != nil {
						return domain.VaultSummary{}, s.summary
					}
					return db.NoteQueries().GetSummary(ctx, v.ID)
				},
				Unembedded: s.unembedded,
				Close:      db.Close,
			}, nil
		},
	}
}

func createNoteVault(t *testing.T) domain.Vault {
	t.Helper()
	return testsupport.NewVault(t, map[string]string{"Bees.md": "# Bees\n\nBees make honey.\n"})
}

func TestScanReportsTheNotesItIndexed(t *testing.T) {
	t.Parallel()
	target := createNoteVault(t)
	s := &scanFixture{unembedded: errors.New("no model here")}
	out, err := runWithDeps(s.deps(t, target), "scan", target.Name)
	if err != nil {
		t.Fatal(err)
	}
	if want := "not embedding " + target.Name + ": no model here\n"; !strings.HasPrefix(out, want) {
		t.Errorf("scan did not say why no vectors are made:\n%q", out)
	}
	for _, want := range []string{"  1 indexed\r", "index now holds 1 notes"} {
		if !strings.Contains(out, want) {
			t.Errorf("scan did not say %q:\n%q", want, out)
		}
	}
	for _, not := range []string{"sources:", "embedded", "vectors made", "extracting "} {
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
	target := createNoteVault(t)
	s := &scanFixture{embedder: constantEmbedder{}}
	out, err := runWithDeps(s.deps(t, target), "scan", target.Name)
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
	target := testsupport.NewVault(t, nil)
	testsupport.WriteBook(t, target.Path, "bee.epub")
	s := &scanFixture{}
	out, err := runWithDeps(s.deps(t, target), "scan", target.Name)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  cutting bee.epub\r", "1 sources: 1 read, 0 unchanged, "} {
		if !strings.Contains(out, want) {
			t.Errorf("scan did not say %q:\n%q", want, out)
		}
	}
	if strings.Contains(out, "cutting \r") {
		t.Errorf("scan named a book when none was open:\n%q", out)
	}
}

func TestScanRebuildsTheIndexWhenAsked(t *testing.T) {
	t.Parallel()
	target := createNoteVault(t)
	s := &scanFixture{}
	for _, args := range [][]string{
		{"scan", target.Name, "--rebuild-index"},
		{"scan", "--rebuild-index", target.Name},
	} {
		s.rebuilt = false
		if _, err := runWithDeps(s.deps(t, target), args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !s.rebuilt {
			t.Errorf("%v did not rebuild the index", args)
		}
	}
}

func TestScanReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	target := createNoteVault(t)

	opening := (&scanFixture{}).deps(t, target)
	opening.Scan = func(context.Context, domain.Vault, bool) (cli.Scan, error) { return cli.Scan{}, errRefused }
	if _, err := runWithDeps(opening, "scan", target.Name); !errors.Is(err, errRefused) {
		t.Errorf("a scan that cannot open answered %v", err)
	}

	summing := &scanFixture{summary: errRefused}
	out, err := runWithDeps(summing.deps(t, target), "scan", target.Name)
	if !errors.Is(err, errRefused) {
		t.Errorf("a scan that cannot count answered %v", err)
	}
	if strings.Contains(out, "index now holds") {
		t.Errorf("scan reported a count it did not have:\n%q", out)
	}

	removed := domain.Vault{ID: target.ID, Name: target.Name, Path: filepath.Join(t.TempDir(), "not-there")}
	if _, err := runWithDeps((&scanFixture{}).deps(t, removed), "scan", removed.Name); err == nil {
		t.Error("a scan of a folder that is not there succeeded")
	}
}

// failing is a model that answers nothing.
type failing struct{ constantEmbedder }

func (failing) Embed(context.Context, []string) ([][]float32, error) { return nil, errRefused }

func TestScanReturnsWhyTheVectorsWereNotMade(t *testing.T) {
	t.Parallel()
	target := createNoteVault(t)
	s := &scanFixture{embedder: failing{}}
	out, err := runWithDeps(s.deps(t, target), "scan", target.Name)
	if !errors.Is(err, errRefused) {
		t.Errorf("a scan whose model failed answered %v", err)
	}
	if strings.Contains(out, "index now holds") {
		t.Errorf("scan reported a count after it failed:\n%q", out)
	}
}

func TestScanNeedsExactlyOneVault(t *testing.T) {
	t.Parallel()
	target := createNoteVault(t)
	for _, args := range [][]string{{"scan"}, {"scan", target.Name, "other"}, {"scan", "--rebuild-index"}} {
		_, err := runWithDeps((&scanFixture{}).deps(t, target), args...)
		if err == nil || !strings.Contains(err.Error(), "usage: numen-cli scan <vault> [--rebuild-index]") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}
