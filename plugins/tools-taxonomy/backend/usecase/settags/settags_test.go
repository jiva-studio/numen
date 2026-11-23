package settags_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/settags"
)

const (
	vaultID = domain.VaultID("garden")
	quay    = "notes/Quay.md"
	dock    = "notes/Dock.md"
	pier    = "notes/Pier.md"
)

// vaultTags reads the tags of the notes of a vault as the host holds them now.
type vaultTags struct {
	vault *hosttest.Vault
}

func (r vaultTags) GetMap(ctx context.Context, _ domain.VaultID) (map[string][]string, error) {
	page, err := r.vault.ReadFrontmatter(ctx, host.FrontmatterQuery{Keys: []string{tags.FrontmatterKey}})
	if err != nil {
		return nil, err
	}
	found := map[string][]string{}
	for _, note := range page.Notes {
		if list := tags.Parse(note.Values[tags.FrontmatterKey]); len(list) > 0 {
			found[note.Path] = list
		}
	}
	return found, nil
}

type scene struct {
	vault   *hosttest.Vault
	useCase *settags.UseCase
}

func newScene(files map[string]string) *scene {
	v := hosttest.New(string(vaultID))
	for path, content := range files {
		v.WriteFile(path, content)
	}
	s := &scene{vault: v}
	s.restart()
	return s
}

func (s *scene) restart() {
	s.useCase = settags.New(vaultTags{vault: s.vault}, func(id domain.VaultID) settags.Vault {
		return settags.Vault{Properties: s.vault.With(id), Store: s.vault.Store()}
	})
}

func (s *scene) execute(t *testing.T, paths, toggles []string) []settags.Result {
	t.Helper()
	results, err := s.useCase.Execute(t.Context(), vaultID, paths, toggles)
	if err != nil {
		t.Fatal(err)
	}
	return results
}

func (s *scene) getTags(t *testing.T, path string) []string {
	t.Helper()
	page, err := s.vault.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Notes) != 1 {
		t.Fatalf("the vault holds %d notes at %s, want one", len(page.Notes), path)
	}
	return tags.Parse(page.Notes[0].Values[tags.FrontmatterKey])
}

func (s *scene) getRecent(t *testing.T) []string {
	t.Helper()
	recent, err := s.useCase.ListRecent(t.Context(), vaultID)
	if err != nil {
		t.Fatal(err)
	}
	return recent
}

func note(frontmatter, body string) string {
	if frontmatter == "" {
		return body + "\n"
	}
	return "---\n" + frontmatter + "\n---\n" + body + "\n"
}

func TestExecuteWritesTheToggledTagsToANoteWithNone(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})

	results := s.execute(t, []string{pier}, []string{"Port", "Sea"})

	if len(results) != 1 || results[0].Path != pier || results[0].Err != nil || !slices.Equal(results[0].Tags, []string{"Port", "Sea"}) {
		t.Errorf("results = %+v, want the tags Port and Sea written to %s", results, pier)
	}
	if got := s.getTags(t, pier); !slices.Equal(got, []string{"Port", "Sea"}) {
		t.Errorf("the note has the tags %q, want Port and Sea", got)
	}
	if content, _ := s.vault.ReadContent(pier); !strings.Contains(content, "The pier.") {
		t.Errorf("the note is %q, want its body kept", content)
	}
}

func TestExecuteClearsOneOfSeveralTagsAndKeepsTheOthersInOrder(t *testing.T) {
	s := newScene(map[string]string{dock: note("tags: [Port, Sea, Quay]", "The dock.")})

	results := s.execute(t, []string{dock}, []string{"Sea"})

	if got := s.getTags(t, dock); !slices.Equal(got, []string{"Port", "Quay"}) {
		t.Errorf("the note has the tags %q, want Port and Quay", got)
	}
	if len(results) != 1 || !slices.Equal(results[0].Tags, []string{"Port", "Quay"}) {
		t.Errorf("results = %+v, want the tags the note has now", results)
	}
}

func TestExecuteSetsOneTagAndClearsAnotherInOneCall(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port, Sea]", "The quay.")})

	s.execute(t, []string{quay}, []string{"Port", "Wharf"})

	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Sea", "Wharf"}) {
		t.Errorf("the note has the tags %q, want Sea and Wharf", got)
	}
}

func TestExecuteClearsATagInAnotherSpelling(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Projects/Web, Sea]", "The quay.")})

	s.execute(t, []string{quay}, []string{"projects/web"})

	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Sea"}) {
		t.Errorf("the note has the tags %q, want Sea alone", got)
	}
}

func TestExecuteRemovesTheKeyWhenNoTagIsLeftAndKeepsTheOtherKeysAndTheBody(t *testing.T) {
	s := newScene(map[string]string{quay: note("type: place\ntags: [Port]", "The quay.")})

	results := s.execute(t, []string{quay}, []string{"Port"})

	page, err := s.vault.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{quay}})
	if err != nil {
		t.Fatal(err)
	}
	if _, hasTags := page.Notes[0].Values[tags.FrontmatterKey]; hasTags || page.Notes[0].Values["type"] != "place" {
		t.Errorf("the frontmatter is %v, want type kept and tags removed", page.Notes[0].Values)
	}
	if content, _ := s.vault.ReadContent(quay); !strings.HasSuffix(content, "---\nThe quay.\n") {
		t.Errorf("the note is %q, want its body kept", content)
	}
	if len(results) != 1 || len(results[0].Tags) != 0 || results[0].Err != nil {
		t.Errorf("results = %+v, want a written note with no tags", results)
	}
}

func TestExecuteSetsATagOnEveryNoteWhenOnlySomeHaveIt(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port, Sea]", "The quay."), pier: note("", "The pier.")})

	s.execute(t, []string{quay, pier}, []string{"Port"})

	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Port", "Sea"}) {
		t.Errorf("the quay has the tags %q, want Port and Sea as before", got)
	}
	if got := s.getTags(t, pier); !slices.Equal(got, []string{"Port"}) {
		t.Errorf("the pier has the tags %q, want Port", got)
	}
}

func TestExecuteClearsATagFromEveryNoteWhenAllHaveIt(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port, Sea]", "The quay."), dock: note("tags: [Port]", "The dock.")})

	s.execute(t, []string{quay, dock}, []string{"Port"})

	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Sea"}) {
		t.Errorf("the quay has the tags %q, want Sea", got)
	}
	if got := s.getTags(t, dock); len(got) != 0 {
		t.Errorf("the dock has the tags %q, want none", got)
	}
}

func TestExecuteDecidesEachTagOfOneCallOnItsOwn(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port, Sea]", "The quay."), pier: note("tags: [Sea]", "The pier.")})

	s.execute(t, []string{quay, pier}, []string{"Port", "Sea"})

	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Port"}) {
		t.Errorf("the quay has the tags %q, want Port", got)
	}
	if got := s.getTags(t, pier); !slices.Equal(got, []string{"Port"}) {
		t.Errorf("the pier has the tags %q, want Port", got)
	}
}

func TestExecuteAnswersEachPathInTheOrderOfTheRequest(t *testing.T) {
	s := newScene(map[string]string{quay: note("", "The quay."), dock: note("", "The dock."), pier: note("", "The pier.")})

	results := s.execute(t, []string{pier, quay, dock}, []string{"Port"})

	var paths []string
	for _, result := range results {
		paths = append(paths, result.Path)
	}
	if !slices.Equal(paths, []string{pier, quay, dock}) {
		t.Errorf("results are for %q, want the order of the request", paths)
	}
}

func TestExecuteWritesTheOtherNotesWhenOneNoteFails(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port]", "The quay."), pier: note("", "The pier.")})
	s.vault.RefuseWriting(quay, errors.New("disk full"))

	results := s.execute(t, []string{quay, pier}, []string{"Sea"})

	if len(results) != 2 || results[0].Err == nil || results[1].Err != nil {
		t.Fatalf("results = %+v, want an error for the quay and none for the pier", results)
	}
	if got := s.getTags(t, quay); !slices.Equal(got, []string{"Port"}) {
		t.Errorf("the quay has the tags %q, want Port as before", got)
	}
	if got := s.getTags(t, pier); !slices.Equal(got, []string{"Sea"}) {
		t.Errorf("the pier has the tags %q, want Sea", got)
	}
}

func TestExecuteReportsANoteThatIsNotThereAndWritesTheOthers(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})

	results := s.execute(t, []string{"notes/Gone.md", pier}, []string{"Sea"})

	if len(results) != 2 || results[0].Err == nil || results[1].Err != nil {
		t.Fatalf("results = %+v, want an error for the missing note and none for the pier", results)
	}
	if got := s.getTags(t, pier); !slices.Equal(got, []string{"Sea"}) {
		t.Errorf("the pier has the tags %q, want Sea", got)
	}
}

func TestExecuteWithNoToggleChangesNothing(t *testing.T) {
	s := newScene(map[string]string{quay: note("tags: [Port]", "The quay.")})
	before, _ := s.vault.ReadContent(quay)

	s.execute(t, []string{quay}, nil)

	if after, _ := s.vault.ReadContent(quay); after != before {
		t.Errorf("the note is %q, want %q as before", after, before)
	}
}

func TestListRecentReturnsTheLatestTagFirst(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})

	s.execute(t, []string{pier}, []string{"Port"})
	s.execute(t, []string{pier}, []string{"Sea"})

	if got := s.getRecent(t); !slices.Equal(got, []string{"Sea", "Port"}) {
		t.Errorf("recent = %q, want Sea then Port", got)
	}
}

func TestListRecentDoesNotCountATagThatWasCleared(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})
	s.execute(t, []string{pier}, []string{"Port"})
	s.execute(t, []string{pier}, []string{"Sea"})

	s.execute(t, []string{pier}, []string{"Port"})

	if got := s.getRecent(t); !slices.Equal(got, []string{"Sea", "Port"}) {
		t.Errorf("recent = %q, want Sea then Port as before the clear", got)
	}
}

func TestListRecentMovesATagSetAgainToTheFront(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})
	s.execute(t, []string{pier}, []string{"Port"})
	s.execute(t, []string{pier}, []string{"Sea"})
	s.execute(t, []string{pier}, []string{"Port"})

	s.execute(t, []string{pier}, []string{"Port"})

	if got := s.getRecent(t); !slices.Equal(got, []string{"Port", "Sea"}) {
		t.Errorf("recent = %q, want Port then Sea", got)
	}
}

func TestListRecentSurvivesARestart(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})
	s.execute(t, []string{pier}, []string{"Port", "Sea"})
	s.restart()

	if got := s.getRecent(t); len(got) != 2 || !slices.Contains(got, "Port") || !slices.Contains(got, "Sea") {
		t.Errorf("recent = %q, want Port and Sea after a restart", got)
	}
}

func TestListRecentIsEmptyBeforeAnyTagIsSet(t *testing.T) {
	s := newScene(map[string]string{pier: note("", "The pier.")})

	if got := s.getRecent(t); len(got) != 0 {
		t.Errorf("recent = %q, want none", got)
	}
}
