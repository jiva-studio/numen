package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// fakeSearchIndex is a test double of the search index: it returns the
// configured passages for a query, or fails with the configured error.
type fakeSearchIndex struct {
	passages []domain.Passage
	err      error
}

func (a fakeSearchIndex) FindLexical(context.Context, domain.VaultID, source.LexicalQuery) ([]domain.Passage, error) {
	return a.passages, a.err
}

func (a fakeSearchIndex) GetNamedPassages(context.Context, domain.VaultID, source.LexicalQuery) ([]domain.Passage, error) {
	return nil, a.err
}

func newVault(folder string) domain.Vault {
	return domain.Vault{ID: "vlt-search", Name: "shelf", Path: folder}
}

func createSearchDeps(folder string, index fakeSearchIndex, words error) stubSession {
	return stubSession{
		Vaults: createVaultList(newVault(folder)),
		Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) {
			return cli.Indexed{Search: cli.Search{
				Search: search.NewByName(registry.NewFind(singleVaultList{newVault(folder)}).Execute, preparing(search.New(index, vaultreader.Readers{}, nil, nil, nil))),
				Words:  words,
			}}, nil
		},
	}
}

func runWithStderr(t *testing.T, deps stubSession, args ...string) (out, stderr string, err error) {
	t.Helper()
	var answer, aside bytes.Buffer
	err = cli.Run(t.Context(), &answer, &aside, args, func(cli.Locations) cli.Session { return deps.session() })
	return answer.String(), aside.String(), err
}

func TestSearchNeedsAVaultAndAQuery(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"search"}, {"search", "shelf"}} {
		_, err := runWithDeps(stubSession{Vaults: createVaultList(getNotebook())}, args...)
		if err == nil || !strings.Contains(err.Error(), "usage: numen-cli search <vault> <query>") {
			t.Errorf("%v answered %v", args, err)
		}
	}
}

func TestSearchReturnsWhatWentWrong(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	for name, provided := range map[string]stubSession{
		"the list of vaults cannot be opened": {
			Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) {
				notFound := func(string) (domain.Vault, error) { return domain.Vault{}, errRefused }
				prepared := preparing(search.New(fakeSearchIndex{}, vaultreader.Readers{}, nil, nil, nil))
				return cli.Indexed{Search: cli.Search{Search: search.NewByName(notFound, prepared)}}, nil
			},
		},
		"the search cannot be opened": {
			Vaults: createVaultList(newVault(folder)),
			Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) {
				return cli.Indexed{}, errRefused
			},
		},
		"the index cannot answer": createSearchDeps(folder, fakeSearchIndex{err: errRefused}, nil),
	} {
		out, err := runWithDeps(provided, "search", "shelf", "bees")
		if !errors.Is(err, errRefused) {
			t.Errorf("%s: search answered %v", name, err)
		}
		if out != "" {
			t.Errorf("%s: search said something before it failed:\n%s", name, out)
		}
	}
}

func TestASearchByWordsAloneReturnsWhy(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	out, copyPath, err := runWithStderr(t, createSearchDeps(folder, fakeSearchIndex{}, errRefused), "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if out != "nothing found\n" {
		t.Errorf("search said %q", out)
	}
	if copyPath != "search by words only: rejected on purpose\n" {
		t.Errorf("search said beside its answer %q", copyPath)
	}

	_, copyPath, err = runWithStderr(t, createSearchDeps(folder, fakeSearchIndex{}, nil), "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if copyPath != "" {
		t.Errorf("a search with a model said beside its answer %q", copyPath)
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

func TestAHalfThatCannotRunIsReportedInTheAdjacentAnswer(t *testing.T) {
	t.Parallel()
	provided := createSearchDeps(t.TempDir(), fakeSearchIndex{}, nil)
	open := provided.Indexed
	provided.Indexed = func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
		options.ErrorHandler(errRefused)
		return open(ctx, options)
	}

	_, stderr, err := runWithStderr(t, provided, "search", "shelf", "bees")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "search by words only: rejected on purpose\n" {
		t.Errorf("search said beside its answer %q", stderr)
	}
}
