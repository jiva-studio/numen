package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// The fake pages the importer's tool is tested against, in addition to videoAddress.
const (
	articleAddress = "https://tides.example/articles/mill-ponds"
	articleTitle   = "Mill ponds of the estuary"
	articleProse   = "A mill pond fills on the flood tide and drives the wheel on the ebb."
	silentAddress  = "https://tides.example/blank"
	brokenAddress  = "https://tides.example/broken"
	// otherArticle is a second article, for a note made in the shown vault.
	otherArticle = "https://tides.example/articles/sluice-gates"
	// callDeadline is how long a tool call may take before the test fails it as waiting for a copy.
	callDeadline = 30 * time.Second
)

// fakeSites returns videoAddress through the mock downloader and the fake pages itself, with
// no network. A copy of a video waits until release is closed.
type fakeSites struct {
	video   link.Fetcher
	release chan struct{}
}

func (s fakeSites) isVideo(address string) bool { return strings.Contains(address, "youtube.com") }

func (s fakeSites) CanCopy(address string) bool {
	return s.isVideo(address) && s.video.CanCopy(address)
}

func (s fakeSites) GetDownloadModel(address string) download.Model {
	if s.isVideo(address) {
		return s.video.GetDownloadModel(address)
	}
	return download.Model{Tool: "invented", Version: "1", Producer: "article"}
}

func (s fakeSites) GetMetadata(ctx context.Context, address string) (download.Metadata, error) {
	if s.isVideo(address) {
		return s.video.GetMetadata(ctx, address)
	}
	return download.Metadata{Title: articleTitle}, nil
}

func (s fakeSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
	switch {
	case s.isVideo(address):
		return s.video.GetText(ctx, address, want)
	case address == silentAddress:
		return download.Text{}, download.ErrNothingDownloaded
	case address == brokenAddress:
		return download.Text{}, errors.New("the server returned 503 Service Unavailable")
	default:
		return download.Text{Producer: "article", Title: articleTitle, Prose: articleProse}, nil
	}
}

func (s fakeSites) Download(ctx context.Context, address string, media download.Media, into io.Writer) (download.Copy, error) {
	select {
	case <-s.release:
	case <-ctx.Done():
		return download.Copy{}, ctx.Err()
	}
	return s.video.Download(ctx, address, media, into)
}

// textVault is the vault whose texts compare versions with the notes it makes, as the host's texts
// read the vault's files.
type textVault struct {
	*hosttest.Vault
	texts *hosttest.Texts
	all   map[domain.VaultID]*hosttest.Texts
}

func (v textVault) Create(ctx context.Context, path string, content []byte) error {
	if err := v.Vault.Create(ctx, path, content); err != nil {
		return err
	}
	if v.texts != nil {
		v.texts.File(path, content)
	}
	return nil
}

func (v textVault) vaults() func(domain.VaultID) link.Vault {
	return func(id domain.VaultID) link.Vault {
		if id == "" {
			return link.Vault{Files: v, Writer: v}
		}
		on := v.With(id)
		return link.Vault{Files: on, Writer: on}
	}
}

func (v textVault) With(id domain.VaultID) textVault {
	return textVault{Vault: v.Vault.With(id), texts: v.all[id], all: v.all}
}

// importFixture is the import tool over the open vault Mills, and the vault Chemistry.
type importFixture struct {
	vault     *hosttest.Vault
	store     *hosttest.Store
	texts     *hosttest.Texts
	chemistry *hosttest.Vault
	chemTexts *hosttest.Texts
	links     *link.Links
	registry  *tools.Registry
}

func newImportFixture(t *testing.T, release chan struct{}) *importFixture {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	chemistry := hosttest.New("Chemistry")
	vault.Beside(chemistry)
	chemTexts := texts.With("Chemistry")
	chemTexts.SetStore(PluginID, chemistry.Store())

	sites := fakeSites{video: download.NewMock(webpage.New()), release: release}
	hosted := link.Host{
		Vaults: textVault{
			Vault: vault, texts: texts, all: map[domain.VaultID]*hosttest.Texts{"Mills": texts, "Chemistry": chemTexts},
		}.vaults(),
		Store: foldersOf(vault), Texts: textsOf(texts), Queue: hosttest.NewQueue(),
		Clock: clock.NewFake(time.Unix(1_700_000_000, 0)),
	}
	links := link.New(hosted,
		func() link.Fetcher { return sites },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	registry, err := NewTools(links).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return &importFixture{
		vault: vault, store: store, texts: texts, chemistry: chemistry, chemTexts: chemTexts,
		links: links, registry: registry,
	}
}

func newUnblockedFixture(t *testing.T) *importFixture {
	t.Helper()
	release := make(chan struct{})
	close(release)
	return newImportFixture(t, release)
}

func invokeAtOnce(t *testing.T, registry *tools.Registry, call tools.Call) *tools.CallResponse {
	t.Helper()
	type answer struct {
		res *tools.CallResponse
		err error
	}
	responses := make(chan answer, 1)
	go func() {
		res, err := registry.InvokeTool(t.Context(), call)
		responses <- answer{res: res, err: err}
	}()
	select {
	case got := <-responses:
		if got.err != nil {
			t.Fatalf("call %s: %v", call.Name, got.err)
		}
		return got.res
	case <-time.After(callDeadline):
		t.Fatalf("%s did not return within %v: the call waits for the copy", call.Name, callDeadline)
		return nil
	}
}

// importArguments is the arguments of url_import with the tool's JSON keys.
type importArguments struct {
	URL           string `json:"url"`
	Folder        string `json:"folder,omitempty"`
	DownloadVideo bool   `json:"downloadVideo,omitempty"`
}

// importAnswer is the result of url_import with the tool's JSON keys.
type importAnswer struct {
	Path           string `json:"path"`
	Title          string `json:"title"`
	Producer       string `json:"producer"`
	Bytes          int    `json:"bytes"`
	HasText        bool   `json:"hasText"`
	Error          string `json:"error"`
	DownloadTaskID string `json:"downloadTaskId"`
	DownloadError  string `json:"downloadError"`
}

func (s *importFixture) callImport(t *testing.T, vault string, in importArguments) *tools.CallResponse {
	t.Helper()
	arguments, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return invokeAtOnce(t, s.registry, tools.Call{VaultID: vault, Name: "url_import", Arguments: arguments})
}

func (s *importFixture) importURL(t *testing.T, vault string, in importArguments) importAnswer {
	t.Helper()
	res := s.callImport(t, vault, in)
	if res.GetResult().GetIsError() {
		t.Fatalf("url_import of %s in %s was rejected: %s", in.URL, vault, res.GetResult().GetText())
	}
	var out importAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func getLinkNotes(t *testing.T, vault *hosttest.Vault) []string {
	t.Helper()
	vault.Reindex()
	paths, err := vault.GetPathsOfType(t.Context(), link.NoteType)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func hasVersion(t *testing.T, texts *hosttest.Texts, path string) bool {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return len(listing.Versions) > 0
}

func TestRegistryListsURLImportOnTheFullServerOnly(t *testing.T) {
	s := newUnblockedFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	infos := listing.Tools
	if len(infos) != 1 || infos[0].GetName() != "url_import" {
		t.Fatalf("the registry lists %+v, want url_import alone", infos)
	}
	if infos[0].GetEffect() != tools.ReadWrite {
		t.Errorf("url_import has effect %v, want ReadWrite", infos[0].GetEffect())
	}
	if !slices.Equal(infos[0].GetServers(), []tools.Server{tools.Full}) {
		t.Errorf("url_import is listed on %v, want the full server alone", infos[0].GetServers())
	}
}

func TestURLImportRequiresOutboundNetwork(t *testing.T) {
	s := newUnblockedFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Tools) != 1 {
		t.Fatalf("the registry lists %+v, want url_import alone", listing.Tools)
	}
	if requires := listing.Tools[0].GetRequires(); !slices.Contains(requires, "link:open") {
		t.Errorf("url_import requires %q, want link:open among them", requires)
	}
}

func TestURLImportMakesTheNoteInTheFolderAndReportsTheArticle(t *testing.T) {
	s := newUnblockedFixture(t)

	out := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "sources/tides"})

	if !strings.HasPrefix(out.Path, "sources/tides/") || !strings.HasSuffix(out.Path, ".md") {
		t.Errorf("the note is at %q, want a note in sources/tides", out.Path)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{out.Path}) {
		t.Errorf("the vault has the link notes %v, want %s alone", notes, out.Path)
	}
	if out.Title != articleTitle || out.Producer != "article" || out.Bytes <= 0 {
		t.Errorf("url_import returned %+v, want the title %q, the producer article and the size of the text", out, articleTitle)
	}
	if !out.HasText || out.Error != "" || out.DownloadTaskID != "" || out.DownloadError != "" {
		t.Errorf("url_import returned %+v, want a fetched text and no copy", out)
	}
	if !hasVersion(t, s.texts, out.Path) {
		t.Errorf("the note %s has no text version", out.Path)
	}
}

func TestURLImportReportsTheCaptionsOfAVideo(t *testing.T) {
	s := newUnblockedFixture(t)

	out := s.importURL(t, "Mills", importArguments{URL: videoAddress})

	if out.Title != "Tidal mill at work" || out.Producer != "captions" || out.Bytes <= 0 || !out.HasText || out.Error != "" {
		t.Errorf("url_import returned %+v, want the captions of the video", out)
	}
	if strings.Contains(out.Path, "/") {
		t.Errorf("the note is at %q, want it at the root with no folder named", out.Path)
	}
}

func TestURLImportReportsAnAddressThatPublishesNothing(t *testing.T) {
	s := newUnblockedFixture(t)

	out := s.importURL(t, "Mills", importArguments{URL: silentAddress})

	if out.HasText || out.Error != "" || out.Bytes != 0 {
		t.Errorf("url_import returned %+v, want nothing published and no failure", out)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{out.Path}) {
		t.Errorf("the vault has the link notes %v, want %s alone", notes, out.Path)
	}
}

func TestURLImportReportsAFailedFetchAndKeepsTheNote(t *testing.T) {
	s := newUnblockedFixture(t)

	res := s.callImport(t, "Mills", importArguments{URL: brokenAddress})

	if res.GetResult().GetIsError() {
		t.Fatalf("a failed fetch was rejected: %s", res.GetResult().GetText())
	}
	var out importAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Error, "503") || out.HasText {
		t.Errorf("url_import returned %+v, want the failure of the fetch", out)
	}
	if _, isKept := s.vault.Content(out.Path); out.Path == "" || !isKept {
		t.Errorf("the note %q is not in the vault after the failed fetch", out.Path)
	}
}

func TestURLImportRejectsNonHTTPAddressAndCreatesNoNote(t *testing.T) {
	s := newUnblockedFixture(t)

	for _, raw := range []string{"not-a-url", "ftp://tides.example/archive", "tides.example/articles"} {
		res := s.callImport(t, "Mills", importArguments{URL: raw})
		if !res.GetResult().GetIsError() {
			t.Errorf("%q returned %s, want an error", raw, res.GetResult().GetStructured())
		}
	}
	if notes := getLinkNotes(t, s.vault); len(notes) != 0 {
		t.Errorf("the vault has the link notes %v, want none", notes)
	}
}

func TestURLImportRejectsACallWithoutAVault(t *testing.T) {
	s := newUnblockedFixture(t)

	res := s.callImport(t, "", importArguments{URL: articleAddress})

	if !res.GetResult().GetIsError() {
		t.Errorf("a call with no vault returned %s, want an error", res.GetResult().GetStructured())
	}
	if notes := getLinkNotes(t, s.vault); len(notes) != 0 {
		t.Errorf("the vault has the link notes %v, want none", notes)
	}
}

func TestURLImportCreatesTheNoteInTheNamedVault(t *testing.T) {
	s := newUnblockedFixture(t)

	inChemistry := s.importURL(t, "Chemistry", importArguments{URL: articleAddress, Folder: "reading"})
	inMills := s.importURL(t, "Mills", importArguments{URL: otherArticle, Folder: "reading"})

	if notes := getLinkNotes(t, s.chemistry); !slices.Equal(notes, []string{inChemistry.Path}) {
		t.Errorf("Chemistry has the link notes %v, want %s alone", notes, inChemistry.Path)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{inMills.Path}) {
		t.Errorf("Mills has the link notes %v, want %s alone", notes, inMills.Path)
	}
	if !hasVersion(t, s.chemTexts, inChemistry.Path) || hasVersion(t, s.texts, inChemistry.Path) {
		t.Errorf("the text of %s is not in Chemistry alone", inChemistry.Path)
	}
	if !hasVersion(t, s.texts, inMills.Path) || hasVersion(t, s.chemTexts, inMills.Path) {
		t.Errorf("the text of %s is not in Mills alone", inMills.Path)
	}
}

func TestURLImportStartsTheDownloadAndReturnsBeforeItEnds(t *testing.T) {
	release := make(chan struct{})
	s := newImportFixture(t, release)

	out := s.importURL(t, "Mills", importArguments{URL: videoAddress, DownloadVideo: true})

	if out.DownloadTaskID != "copy:Mills:"+out.Path || out.DownloadError != "" {
		t.Errorf("url_import returned %+v, want the copy task copy:Mills:%s", out, out.Path)
	}
	if hasCopy(t, s.store) {
		t.Error("the copy is in the plugin's folder before the download was let through")
	}
	close(release)
	s.links.Wait()
	if !hasCopy(t, s.store) {
		t.Error("the copy is not in the plugin's folder once the background copy ended")
	}
}

func TestURLImportDownloadsInTwoVaultsOnTheSamePath(t *testing.T) {
	release := make(chan struct{})
	s := newImportFixture(t, release)

	inMills := s.importURL(t, "Mills", importArguments{URL: videoAddress, DownloadVideo: true})
	inChemistry := s.importURL(t, "Chemistry", importArguments{URL: videoAddress, DownloadVideo: true})

	if inMills.Path != inChemistry.Path {
		t.Fatalf("paths %q and %q, want the same path in both vaults", inMills.Path, inChemistry.Path)
	}
	if inMills.DownloadTaskID != "copy:Mills:"+inMills.Path || inChemistry.DownloadTaskID != "copy:Chemistry:"+inChemistry.Path {
		t.Errorf("task ids %q and %q, want copy:Mills:%s and copy:Chemistry:%s",
			inMills.DownloadTaskID, inChemistry.DownloadTaskID, inMills.Path, inChemistry.Path)
	}
	close(release)
	s.links.Wait()
	if !hasCopy(t, s.store) || !hasCopy(t, s.chemistry.Store()) {
		t.Error("one of the two vaults has no copy after both downloads ended")
	}
}

func TestURLImportDownloadsOnlyWhenRequested(t *testing.T) {
	s := newUnblockedFixture(t)

	out := s.importURL(t, "Mills", importArguments{URL: videoAddress})
	s.links.Wait()

	if out.DownloadTaskID != "" || out.DownloadError != "" {
		t.Errorf("url_import returned %+v, want no copy", out)
	}
	if hasCopy(t, s.store) {
		t.Error("a video was copied without the copy argument")
	}
}

func TestURLImportReportsAnAddressNothingCopies(t *testing.T) {
	s := newUnblockedFixture(t)

	out := s.importURL(t, "Mills", importArguments{URL: articleAddress, DownloadVideo: true})
	s.links.Wait()

	if out.DownloadError == "" || out.DownloadTaskID != "" {
		t.Errorf("url_import returned %+v, want the reason no copy was started", out)
	}
	if out.Title != articleTitle || out.Error != "" {
		t.Errorf("url_import returned %+v, want the article fetched", out)
	}
}

func TestTheImportToolHasItsInstructions(t *testing.T) {
	s := newUnblockedFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	sentences := getToolSentences(listing, "url_import")
	if len(sentences) == 0 {
		t.Fatal("url_import lists no instructions")
	}
	if !strings.Contains(strings.Join(sentences, " "), "url_import") {
		t.Errorf("no instruction of url_import names it: %q", sentences)
	}
	for _, sentence := range sentences {
		if !isOneSentence(sentence) {
			t.Errorf("an instruction of url_import is not one sentence: %q", sentence)
		}
	}
}
