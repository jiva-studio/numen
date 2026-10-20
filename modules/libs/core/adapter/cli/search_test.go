package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// fakeSearchIndex is what the index of a search says: the passages it names for a
// query, or the error it fails with.
type fakeSearchIndex struct {
	passages []domain.Passage
	err      error
}

func (a fakeSearchIndex) FindLexical(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return a.passages, a.err
}

func (a fakeSearchIndex) GetNamedPassages(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return nil, a.err
}

func (a fakeSearchIndex) FindNearest(context.Context, domain.VaultID, string, []float32, []domain.SourceKind, int, float64) ([]domain.Passage, error) {
	return nil, a.err
}

func newVault(folder string) domain.Vault {
	return domain.Vault{ID: "vlt-search", Name: "shelf", Path: folder}
}

// createSearchDeps is a terminal whose only vault is the folder given and whose
// index answers as given. words is why the search runs by words alone.
func createSearchDeps(folder string, index fakeSearchIndex, words error) cli.Deps {
	return cli.Deps{
		Vaults: createVaultList(newVault(folder)),
		Search: func(context.Context, port.ErrorHandler) (cli.Search, error) {
			return cli.Search{
				Search: search.New(index, filesystem.VaultReaders{}, nil, nil, nil, 0, nil),
				Words:  words,
			}, nil
		},
	}
}

// runWithStderr runs the terminal and keeps what it said beside its answer.
func runWithStderr(t *testing.T, given cli.Deps, args ...string) (out, beside string, err error) {
	var answer, aside bytes.Buffer
	t.Helper()
	err = cli.Run(t.Context(), &answer, &aside, args, func(cli.Locations) cli.Deps { return given })
	return answer.String(), aside.String(), err
}

func TestSearchNeedsAVaultAndAQuery(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"search"}, {"search", "shelf"}} {
		_, err := runWithDeps(cli.Deps{Vaults: createVaultList(notebook)}, args...)
		if err == nil || !strings.Contains(err.Error(), "usage: numen-cli search <vault> <query>") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}

func TestSearchReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	for name, given := range map[string]cli.Deps{
		"the list of vaults cannot be opened": {
			Vaults: func() (cli.Vaults, error) { return cli.Vaults{}, errRefused },
		},
		"the search cannot be opened": {
			Vaults: createVaultList(newVault(folder)),
			Search: func(context.Context, port.ErrorHandler) (cli.Search, error) { return cli.Search{}, errRefused },
		},
		"the index cannot answer": createSearchDeps(folder, fakeSearchIndex{err: errRefused}, nil),
	} {
		out, err := runWithDeps(given, "search", "shelf", "bees")
		if !errors.Is(err, errRefused) {
			t.Errorf("%s: search answered %v", name, err)
		}
		if out != "" {
			t.Errorf("%s: search said something before it failed:\n%s", name, out)
		}
	}
}

func TestASearchByWordsAloneSaysWhy(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	out, beside, err := runWithStderr(t, createSearchDeps(folder, fakeSearchIndex{}, errRefused), "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if out != "nothing found\n" {
		t.Errorf("search said %q", out)
	}
	if beside != "searching by words alone: refused on purpose\n" {
		t.Errorf("search said beside its answer %q", beside)
	}

	_, beside, err = runWithStderr(t, createSearchDeps(folder, fakeSearchIndex{}, nil), "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if beside != "" {
		t.Errorf("a search with a model said beside its answer %q", beside)
	}
}

func TestEachPassageIsNamedByItsSourceAndItsPlace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	folder := testsupport.TempDir(t)
	for name, body := range map[string]string{
		"Bees.md":  "Bees fly\n\n   far",
		"Hives.md": "Hives  hum",
	} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	index := fakeSearchIndex{passages: []domain.Passage{
		{ChunkID: "c1", Source: "Bees.md", Kind: domain.KindNote, Length: len("Bees fly\n\n   far"), Location: "p. 3"},
		{ChunkID: "c2", Source: "Hives.md", Kind: domain.KindNote, Length: len("Hives  hum")},
	}}

	out, _, err := runWithStderr(t, createSearchDeps(folder, index, nil), "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Bees.md · p. 3\n  Bees fly far\nHives.md\n  Hives hum\n"; out != want {
		t.Errorf("search said\n%q\nwant\n%q", out, want)
	}
}

func TestAnExcerptIsOneLineOfAtMostOneHundredSixtyCharacters(t *testing.T) {
	t.Parallel()
	whole := strings.Repeat("é", 160)
	for name, c := range map[string]struct{ text, want string }{
		"short text with the shape of a file": {"  one\n\ttwo   three \n", "one two three"},
		"exactly as much as a line takes":     {whole, whole},
		"one character more":                  {whole + "é", whole + "…"},
		"far more":                            {strings.Repeat("word ", 100), strings.TrimSpace(strings.Repeat("word ", 100))[:160] + "…"},
	} {
		if got := cli.Excerpt(c.text); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestAHalfThatCannotRunIsHeardAboutBesideTheAnswer(t *testing.T) {
	t.Parallel()
	given := createSearchDeps(t.TempDir(), fakeSearchIndex{}, nil)
	open := given.Search
	given.Search = func(ctx context.Context, errorHandler port.ErrorHandler) (cli.Search, error) {
		errorHandler(errRefused)
		return open(ctx, errorHandler)
	}

	_, beside, err := runWithStderr(t, given, "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if beside != "answering by words alone: refused on purpose\n" {
		t.Errorf("search said beside its answer %q", beside)
	}
}
