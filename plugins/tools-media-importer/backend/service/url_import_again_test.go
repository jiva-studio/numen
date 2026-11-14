package service

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// revisedProse is what articleAddress publishes from the second fetch of its text on.
const revisedProse = "The mill pond was dredged in spring, and the wheel now turns on both tides."

// revisedSites is fakeSites whose article publishes revisedProse from the second fetch on, or fails
// with fetchErr where it is set.
type revisedSites struct {
	fakeSites
	mu       sync.Mutex
	fetches  int
	fetchErr error
}

func (s *revisedSites) GetText(ctx context.Context, address string, want fetch.PreferredCaptions) (fetch.Text, error) {
	text, err := s.fakeSites.GetText(ctx, address, want)
	if err != nil || address != articleAddress {
		return text, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fetches++
	if s.fetches > 1 && s.fetchErr != nil {
		return fetch.Text{}, s.fetchErr
	}
	if s.fetches > 1 {
		text.Prose = revisedProse
	}
	return text, nil
}

func newRevisedFixture(t *testing.T) *importFixture {
	t.Helper()
	return newRefetchFixture(t, nil)
}

func newRefetchFixture(t *testing.T, fetchErr error) *importFixture {
	t.Helper()
	release := make(chan struct{})
	close(release)
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	sites := &revisedSites{fakeSites: fakeSites{video: download.NewMock(webpage.New()), release: release}, fetchErr: fetchErr}
	hosted := link.Host{
		Vaults: textVault{Vault: vault, texts: texts, all: map[domain.VaultID]*hosttest.Texts{"Mills": texts}}.vaults(),
		Store:  foldersOf(vault), Texts: textsOf(texts), Queue: hosttest.NewQueue(),
		Clock:   newFakeClock(time.Unix(1_700_000_000, 0)),
		Entropy: random.Entropy{},
	}
	links := link.New(hosted,
		func() ports.Fetcher { return sites },
		func() fetch.Settings { return fetch.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	registry, err := NewTools(links).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return &importFixture{vault: vault, store: store, texts: texts, links: links, registry: registry}
}

func readLiveText(t *testing.T, texts *hosttest.Texts, filePath string) string {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), filePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		whole, err := host.ReadWholeText(t.Context(), texts, version)
		if err != nil {
			t.Fatal(err)
		}
		return whole.Text
	}
	t.Fatalf("%s has no live text version: %+v", filePath, listing)
	return ""
}

func decodeImportAnswer(t *testing.T, res *tools.CallResponse) importAnswer {
	t.Helper()
	var out importAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func readNoteID(filePath, content string) string {
	return markdown.Parse(domain.FileStat{Path: filePath}, []byte(content)).ID
}

func TestURLImportOfSameAddressToSameFolderUpdatesLinkNote(t *testing.T) {
	s := newRevisedFixture(t)
	first := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "sources"})
	before, isThere := s.vault.Content(first.Path)
	if !isThere {
		t.Fatalf("the first import made no note at %s", first.Path)
	}
	if text := readLiveText(t, s.texts, first.Path); !strings.Contains(text, articleProse) {
		t.Fatalf("the first import recorded the text %q, want it to contain %q", text, articleProse)
	}

	res := s.callImport(t, "Mills", importArguments{URL: articleAddress, Folder: "sources"})

	if res.GetResult().GetIsError() {
		t.Fatalf("the second url_import of %s into sources was rejected: %s", articleAddress, res.GetResult().GetText())
	}
	second := decodeImportAnswer(t, res)
	if second.Path != first.Path || second.HasText != first.HasText || !second.HasText {
		t.Errorf("the second import returned %+v, want the path %s and hasText %v of the first", second, first.Path, first.HasText)
	}
	if second.Title != first.Title || second.Producer != first.Producer || second.Error != "" {
		t.Errorf("the second import returned %+v, want the title and the producer of the first %+v and no error", second, first)
	}
	after, isKept := s.vault.Content(first.Path)
	if !isKept || after != before {
		t.Errorf("the note %s reads %q after the second import, want it unchanged as %q", first.Path, after, before)
	}
	if id := readNoteID(first.Path, after); id == "" || id != readNoteID(first.Path, before) {
		t.Errorf("the note has the id %q after the second import, want %q", id, readNoteID(first.Path, before))
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{first.Path}) {
		t.Errorf("the vault has the link notes %v, want %s alone", notes, first.Path)
	}
	text := readLiveText(t, s.texts, first.Path)
	if !strings.Contains(text, revisedProse) || strings.Contains(text, articleProse) {
		t.Errorf("the live text of %s is %q, want the newly fetched text %q", first.Path, text, revisedProse)
	}
	if second.Bytes != len(text) {
		t.Errorf("the second import returned bytes %d, want %d, the size of the newly fetched text", second.Bytes, len(text))
	}
}

func TestURLImportOfSameAddressToOtherFolderReturnsTheNoteItHas(t *testing.T) {
	s := newUnblockedFixture(t)
	first := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "reading"})

	res := s.callImport(t, "Mills", importArguments{URL: articleAddress, Folder: "archive"})

	if res.GetResult().GetIsError() {
		t.Fatalf("url_import of %s into archive was rejected: %s", articleAddress, res.GetResult().GetText())
	}
	second := decodeImportAnswer(t, res)
	if second.Path != first.Path || !second.HasText {
		t.Errorf("url_import into archive returned %+v, want the note %s with its text", second, first.Path)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{first.Path}) {
		t.Errorf("the vault has the link notes %v, want %s alone", notes, first.Path)
	}
}

func TestURLImportOntoFileThatIsNotItsLinkNoteIsRejected(t *testing.T) {
	for name, content := range map[string]string{
		"a note of its own":              "---\ntitle: Pond survey\n---\nWater level measured at the sluice.\n",
		"a link note of another address": "---\ntype: link\nurl: " + otherArticle + "\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			s := newUnblockedFixture(t)
			probe := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "probe"})
			targetPath := path.Join("desk", path.Base(probe.Path))
			s.vault.Remove(probe.Path)
			s.vault.File(targetPath, content)

			res := s.callImport(t, "Mills", importArguments{URL: articleAddress, Folder: "desk"})

			if !res.GetResult().GetIsError() {
				t.Fatalf("url_import onto %s returned %s, want an error", targetPath, res.GetResult().GetStructured())
			}
			if text := res.GetResult().GetText(); !strings.Contains(text, targetPath) {
				t.Errorf("the error %q does not name the path %s", text, targetPath)
			}
			if after, _ := s.vault.Content(targetPath); after != content {
				t.Errorf("%s reads %q after the rejected import, want it unchanged as %q", targetPath, after, content)
			}
		})
	}
}

func TestURLImportWhoseFetchFailsReportsTheTextTheNoteHas(t *testing.T) {
	s := newRefetchFixture(t, errors.New("the site is down"))
	first := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "sources"})
	if !first.HasText {
		t.Fatalf("the first import returned %+v, want hasText", first)
	}

	second := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "sources"})

	if second.Error == "" {
		t.Errorf("the import whose fetch failed returned no error: %+v", second)
	}
	text := readLiveText(t, s.texts, first.Path)
	if !second.HasText || second.Bytes != len(text) || second.Title != first.Title {
		t.Errorf("the import whose fetch failed returned %+v, want hasText, bytes %d and the title %q of the text the note has", second, len(text), first.Title)
	}
}
