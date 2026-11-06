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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// The invented pages the importer's tool is tested against, besides videoAddress.
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

// inventedSites answers videoAddress through the mock downloader and the invented pages itself, with
// no network. A copy of a video waits until release is closed.
type inventedSites struct {
	video   link.Fetcher
	release chan struct{}
}

func (s inventedSites) isVideo(address string) bool { return strings.Contains(address, "youtube.com") }

func (s inventedSites) CanCopy(address string) bool {
	return s.isVideo(address) && s.video.CanCopy(address)
}

func (s inventedSites) GetDownloadModel(address string) download.Model {
	if s.isVideo(address) {
		return s.video.GetDownloadModel(address)
	}
	return download.Model{Tool: "invented", Version: "1", Producer: "article"}
}

func (s inventedSites) GetMetadata(ctx context.Context, address string) (download.Metadata, error) {
	if s.isVideo(address) {
		return s.video.GetMetadata(ctx, address)
	}
	return download.Metadata{Title: articleTitle}, nil
}

func (s inventedSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
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

func (s inventedSites) Download(ctx context.Context, address string, into io.Writer) (download.Copy, error) {
	select {
	case <-s.release:
	case <-ctx.Done():
		return download.Copy{}, ctx.Err()
	}
	return s.video.Download(ctx, address, into)
}

// notedVault is the vault whose texts compare versions with the notes it makes, as the host's texts
// read the vault's files.
type notedVault struct {
	host.Vault
	// texts is the texts of this vault, nil for a vault the scene does not hold.
	texts *hosttest.Texts
	// all is the texts of every vault of the scene, by identity.
	all map[domain.VaultID]*hosttest.Texts
}

func (v notedVault) Create(ctx context.Context, path string, content []byte) error {
	if err := v.Vault.Create(ctx, path, content); err != nil {
		return err
	}
	if v.texts != nil {
		v.texts.File(path, content)
	}
	return nil
}

func (v notedVault) With(id domain.VaultID) host.Vault {
	return notedVault{Vault: v.Vault.With(id), texts: v.all[id], all: v.all}
}

// importScene is the import tool over the vault Mills, shown, and the vault Chemistry beside it.
type importScene struct {
	vault     *hosttest.Vault
	store     *hosttest.Store
	texts     *hosttest.Texts
	chemistry *hosttest.Vault
	chemTexts *hosttest.Texts
	links     *link.Links
	registry  *tools.Registry
}

// newImportScene is the scene whose copies wait until release is closed.
func newImportScene(t *testing.T, release chan struct{}) *importScene {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	chemistry := hosttest.New("Chemistry")
	vault.Beside(chemistry)
	chemTexts, isTexts := texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of Chemistry are not hosttest texts")
	}
	chemTexts.SetStore(PluginID, chemistry.Store())

	sites := inventedSites{video: download.NewMock(), release: release}
	hosted := link.Host{
		Vault: notedVault{
			Vault: vault, texts: texts, all: map[domain.VaultID]*hosttest.Texts{"Mills": texts, "Chemistry": chemTexts},
		},
		Store: store, Texts: texts, Queue: hosttest.NewQueue(),
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
	return &importScene{
		vault: vault, store: store, texts: texts, chemistry: chemistry, chemTexts: chemTexts,
		links: links, registry: registry,
	}
}

// newReleasedScene is the scene whose copies run at once.
func newReleasedScene(t *testing.T) *importScene {
	t.Helper()
	release := make(chan struct{})
	close(release)
	return newImportScene(t, release)
}

// invokeAtOnce calls the tool and fails the test when the call does not answer within callDeadline.
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
		t.Fatalf("%s did not answer within %v: the call waits for the copy", call.Name, callDeadline)
		return nil
	}
}

func (s *importScene) callImport(t *testing.T, vault string, in URLImportInput) *tools.CallResponse {
	t.Helper()
	arguments, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return invokeAtOnce(t, s.registry, tools.Call{VaultID: vault, Name: "url_import", Arguments: arguments})
}

func (s *importScene) importURL(t *testing.T, vault string, in URLImportInput) URLImportOutput {
	t.Helper()
	res := s.callImport(t, vault, in)
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("url_import of %s in %s was refused: %s", in.URL, vault, res.GetResult().GetText())
	}
	var out URLImportOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

// getLinkNotes is the link notes the vault holds.
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
	s := newReleasedScene(t)

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

func TestTheInstructionsAreOneParagraphNamingURLImport(t *testing.T) {
	if !strings.Contains(instructions, "url_import") {
		t.Errorf("the instructions do not name url_import: %q", instructions)
	}
	if strings.Contains(strings.TrimSpace(instructions), "\n\n") {
		t.Errorf("the instructions are more than one paragraph: %q", instructions)
	}
}

func TestURLImportMakesTheNoteInTheFolderAndReportsTheArticle(t *testing.T) {
	s := newReleasedScene(t)

	out := s.importURL(t, "Mills", URLImportInput{URL: articleAddress, Folder: "sources/tides"})

	if !strings.HasPrefix(out.Path, "sources/tides/") || !strings.HasSuffix(out.Path, ".md") {
		t.Errorf("the note is at %q, want a note in sources/tides", out.Path)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{out.Path}) {
		t.Errorf("the vault holds the link notes %v, want %s alone", notes, out.Path)
	}
	if out.Title != articleTitle || out.Producer != "article" || out.Bytes <= 0 {
		t.Errorf("url_import returned %+v, want the title %q, the producer article and the size of the text", out, articleTitle)
	}
	if out.IsNothing || out.Error != "" || out.CopyTaskID != "" || out.CopyError != "" {
		t.Errorf("url_import returned %+v, want a fetched text and no copy", out)
	}
	if !hasVersion(t, s.texts, out.Path) {
		t.Errorf("the note %s has no text version", out.Path)
	}
}

func TestURLImportReportsTheCaptionsOfAVideo(t *testing.T) {
	s := newReleasedScene(t)

	out := s.importURL(t, "Mills", URLImportInput{URL: videoAddress})

	if out.Title != "Tidal mill at work" || out.Producer != "captions" || out.Bytes <= 0 || out.IsNothing || out.Error != "" {
		t.Errorf("url_import returned %+v, want the captions of the video", out)
	}
	if strings.Contains(out.Path, "/") {
		t.Errorf("the note is at %q, want it at the root with no folder named", out.Path)
	}
}

func TestURLImportReportsAnAddressThatPublishesNothing(t *testing.T) {
	s := newReleasedScene(t)

	out := s.importURL(t, "Mills", URLImportInput{URL: silentAddress})

	if !out.IsNothing || out.Error != "" || out.Bytes != 0 {
		t.Errorf("url_import returned %+v, want nothing published and no failure", out)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{out.Path}) {
		t.Errorf("the vault holds the link notes %v, want %s alone", notes, out.Path)
	}
}

func TestURLImportReportsAFailedFetchAndKeepsTheNote(t *testing.T) {
	s := newReleasedScene(t)

	res := s.callImport(t, "Mills", URLImportInput{URL: brokenAddress})

	if res.GetResult().GetIsRefusal() {
		t.Fatalf("a failed fetch was refused: %s", res.GetResult().GetText())
	}
	var out URLImportOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Error, "503") || out.IsNothing {
		t.Errorf("url_import returned %+v, want the failure of the fetch", out)
	}
	if _, isKept := s.vault.Content(out.Path); out.Path == "" || !isKept {
		t.Errorf("the note %q is not in the vault after the failed fetch", out.Path)
	}
}

func TestURLImportRefusesWhatIsNoWebAddressAndMakesNoNote(t *testing.T) {
	s := newReleasedScene(t)

	for _, raw := range []string{"not-a-url", "ftp://tides.example/archive", "tides.example/articles"} {
		res := s.callImport(t, "Mills", URLImportInput{URL: raw})
		if !res.GetResult().GetIsRefusal() {
			t.Errorf("%q returned %s, want a refusal", raw, res.GetResult().GetStructured())
		}
	}
	if notes := getLinkNotes(t, s.vault); len(notes) != 0 {
		t.Errorf("the vault holds the link notes %v, want none", notes)
	}
}

func TestURLImportRefusesACallThatNamesNoVault(t *testing.T) {
	s := newReleasedScene(t)

	res := s.callImport(t, "", URLImportInput{URL: articleAddress})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a call with no vault returned %s, want a refusal", res.GetResult().GetStructured())
	}
	if notes := getLinkNotes(t, s.vault); len(notes) != 0 {
		t.Errorf("the vault holds the link notes %v, want none", notes)
	}
}

func TestURLImportMakesTheNoteInTheVaultTheCallNames(t *testing.T) {
	s := newReleasedScene(t)

	inChemistry := s.importURL(t, "Chemistry", URLImportInput{URL: articleAddress, Folder: "reading"})
	inMills := s.importURL(t, "Mills", URLImportInput{URL: otherArticle, Folder: "reading"})

	if notes := getLinkNotes(t, s.chemistry); !slices.Equal(notes, []string{inChemistry.Path}) {
		t.Errorf("Chemistry holds the link notes %v, want %s alone", notes, inChemistry.Path)
	}
	if notes := getLinkNotes(t, s.vault); !slices.Equal(notes, []string{inMills.Path}) {
		t.Errorf("Mills holds the link notes %v, want %s alone", notes, inMills.Path)
	}
	if !hasVersion(t, s.chemTexts, inChemistry.Path) || hasVersion(t, s.texts, inChemistry.Path) {
		t.Errorf("the text of %s is not in Chemistry alone", inChemistry.Path)
	}
	if !hasVersion(t, s.texts, inMills.Path) || hasVersion(t, s.chemTexts, inMills.Path) {
		t.Errorf("the text of %s is not in Mills alone", inMills.Path)
	}
}

func TestURLImportStartsTheCopyAndAnswersBeforeItEnds(t *testing.T) {
	release := make(chan struct{})
	s := newImportScene(t, release)

	out := s.importURL(t, "Mills", URLImportInput{URL: videoAddress, ShouldCopy: true})

	if out.CopyTaskID != "copy:"+out.Path || out.CopyError != "" {
		t.Errorf("url_import returned %+v, want the copy task copy:%s", out, out.Path)
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

func TestURLImportMakesNoCopyUnlessAsked(t *testing.T) {
	s := newReleasedScene(t)

	out := s.importURL(t, "Mills", URLImportInput{URL: videoAddress})
	s.links.Wait()

	if out.CopyTaskID != "" || out.CopyError != "" {
		t.Errorf("url_import returned %+v, want no copy", out)
	}
	if hasCopy(t, s.store) {
		t.Error("a video was copied with copy not asked")
	}
}

func TestURLImportReportsAnAddressNothingCopies(t *testing.T) {
	s := newReleasedScene(t)

	out := s.importURL(t, "Mills", URLImportInput{URL: articleAddress, ShouldCopy: true})
	s.links.Wait()

	if out.CopyError == "" || out.CopyTaskID != "" {
		t.Errorf("url_import returned %+v, want the reason no copy was started", out)
	}
	if out.Title != articleTitle || out.Error != "" {
		t.Errorf("url_import returned %+v, want the article fetched", out)
	}
}

func TestRegistryListsTheURLImportInstructions(t *testing.T) {
	s := newReleasedScene(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions == "" || listing.Instructions != instructions {
		t.Errorf("the registry lists the instructions %q, want %q", listing.Instructions, instructions)
	}
	if !strings.Contains(listing.Instructions, "url_import") {
		t.Errorf("the listed instructions do not name url_import: %q", listing.Instructions)
	}
	if strings.Contains(strings.TrimSpace(listing.Instructions), "\n\n") {
		t.Errorf("the listed instructions are more than one paragraph: %q", listing.Instructions)
	}
}
