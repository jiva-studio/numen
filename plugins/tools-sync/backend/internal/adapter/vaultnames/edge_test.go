package vaultnames_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/vaultnames"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	vaultID    domain.VaultID = "garden"
	plain      string         = "Plan.md"
	composed   string         = "\u0419ога и \u0451ж.md"
	decomposed string         = "\u0418\u0306ога и \u0435\u0308ж.md"
)

func listPaths(t *testing.T, files ports.Files) []string {
	t.Helper()
	page, err := files.ListFileChanges(t.Context(), host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: "", Seq: 0}, Events: []string{host.EventFileAdded}, Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	page, err = files.ListFileChanges(t.Context(), host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: page.Next.FeedID, Seq: 0}, Events: []string{host.EventFileAdded}, Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, change := range page.Changes {
		paths = append(paths, change.Path)
	}
	return paths
}

func TestFiles_ListsADecomposedNameAsTheComposedName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))

	paths := listPaths(t, files)

	if len(paths) != 1 || paths[0] != composed {
		t.Errorf("listed %q, want only %q", paths, composed)
	}
}

func TestFiles_ReadsAFileOfADecomposedNameByItsComposedNameAfterARestart(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))

	content, stat, err := files.Read(t.Context(), composed)

	if err != nil || string(content) != "one" || stat.Path != composed {
		t.Errorf("read = %q at %q, %v, want one at %q", content, stat.Path, err, composed)
	}
}

func TestFiles_WritesOverTheFileOfADecomposedNameAndMakesNoSecondFile(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))
	_, stat, err := files.Read(t.Context(), composed)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := files.Write(t.Context(), composed, []byte("two"), stat); err != nil {
		t.Fatal(err)
	}

	if got, _ := vault.ReadContent(decomposed); got != "two" {
		t.Errorf("the file of the decomposed name holds %q, want two", got)
	}
	if _, isThere := vault.ReadContent(composed); isThere {
		t.Error("a file of the composed name was made next to the file of the decomposed name")
	}
}

func TestFiles_RefusesToCreateAFileOfTheComposedNameOverOneOfTheDecomposedName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))

	err := files.Create(t.Context(), composed, []byte("two"))

	if !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("create = %v, want %v", err, host.ErrAlreadyExists)
	}
	if _, isThere := vault.ReadContent(composed); isThere {
		t.Error("a file of the composed name was made next to the file of the decomposed name")
	}
}

func TestFiles_FollowsAFileRemadeWithTheComposedName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))
	if _, err := files.StatFile(t.Context(), composed); err != nil {
		t.Fatal(err)
	}
	vault.Remove(decomposed)
	vault.WriteFile(composed, "two")

	content, _, err := files.Read(t.Context(), composed)

	if err != nil || string(content) != "two" {
		t.Errorf("read = %q, %v, want two", content, err)
	}
}

func TestFiles_ReportsAFileThatIsNotThereAsNotThere(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	files := vaultnames.New().GetFiles(vaultID, vault.With(vaultID))

	_, err := files.StatFile(t.Context(), composed)

	if !errors.Is(err, host.ErrNotExist) {
		t.Errorf("stat = %v, want %v", err, host.ErrNotExist)
	}
}

func TestFiles_DoesNotReadTheFeedForAPlainName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(plain, "one")
	spy := &feedSpy{Files: vault.With(vaultID)}
	files := vaultnames.New().GetFiles(vaultID, spy)

	if _, err := files.StatFile(t.Context(), "Missing.md"); !errors.Is(err, host.ErrNotExist) {
		t.Fatal(err)
	}
	if err := files.Create(t.Context(), "Made.md", []byte("x")); err != nil {
		t.Fatal(err)
	}

	if spy.reads != 0 {
		t.Errorf("the feed was read %d times for plain names, want none", spy.reads)
	}
}

type feedSpy struct {
	ports.Files
	reads int
}

func (s *feedSpy) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	s.reads++
	return s.Files.ListFileChanges(ctx, q)
}

func TestTransfers_ReadsMovesAndRemovesAFileOfADecomposedNameByItsComposedName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	edge := vaultnames.New()
	files := edge.GetFiles(vaultID, vault.With(vaultID))
	transfers := edge.GetTransfers(vaultID, files, vault.With(vaultID))

	part, err := transfers.ReadBytes(t.Context(), composed, 0, 10)
	if err != nil || string(part.Content) != "one" || part.Stat.Path != composed {
		t.Fatalf("read = %q at %q, %v, want one at %q", part.Content, part.Stat.Path, err, composed)
	}
	if _, err := transfers.MoveFile(t.Context(), composed, "Moved.md", nil); err != nil {
		t.Fatalf("move: %v", err)
	}
	if got, _ := vault.ReadContent("Moved.md"); got != "one" {
		t.Errorf("the moved file holds %q, want one", got)
	}
}

func TestTransfers_RemovesAFileOfADecomposedNameByItsComposedName(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	vault.WriteFile(decomposed, "one")
	edge := vaultnames.New()
	files := edge.GetFiles(vaultID, vault.With(vaultID))
	transfers := edge.GetTransfers(vaultID, files, vault.With(vaultID))

	if _, err := transfers.RemoveFile(t.Context(), composed, nil); err != nil {
		t.Fatal(err)
	}

	if _, isThere := vault.ReadContent(decomposed); isThere {
		t.Error("the file of the decomposed name is still there")
	}
}
