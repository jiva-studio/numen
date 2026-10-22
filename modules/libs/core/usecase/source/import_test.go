package source

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/operation"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A site that answers with what a test puts in it, and counts what it was asked
// for: nothing here reaches a network.
type site struct {
	title  string
	length int
	cues   []domain.TranscriptCue
	prose  string
	bytes  []byte
	asked  []string
	// inOperation records, for each Metadata and Download, whether it ran under an operation.
	inOperation []bool
	refusing    error
}

func (s *site) GetDownloadModel(_ context.Context, _ domain.URL) (port.DownloadModel, error) {
	producer := domain.Captions
	if len(s.cues) == 0 && s.prose != "" {
		producer = domain.Article
	}
	return port.DownloadModel{Tool: "a test", Version: "1", Producer: producer}, nil
}

func (s *site) GetMetadata(ctx context.Context, at domain.URL) (port.Metadata, error) {
	s.inOperation = append(s.inOperation, operation.IsActive(ctx))
	s.asked = append(s.asked, "metadata "+string(at))
	if s.refusing != nil {
		return port.Metadata{}, s.refusing
	}
	return port.Metadata{Title: s.title, Length: s.length}, nil
}

func (s *site) GetText(
	_ context.Context, at domain.URL, _ port.PreferredCaptions,
) (port.Text, error) {
	s.asked = append(s.asked, "text "+string(at))
	if len(s.cues) > 0 {
		return port.Text{
			Producer: domain.Captions, Cues: s.cues, Title: s.title, Length: s.length,
		}, nil
	}
	if s.prose != "" {
		return port.Text{Producer: domain.Article, Prose: s.prose, Title: s.title}, nil
	}
	return port.Text{}, port.ErrNothingDownloaded
}

func (s *site) Download(ctx context.Context, at domain.URL, into io.Writer) (port.Copy, error) {
	s.inOperation = append(s.inOperation, operation.IsActive(ctx))
	s.asked = append(s.asked, "download "+string(at))
	if len(s.bytes) == 0 {
		return port.Copy{}, port.ErrNothingDownloaded
	}
	if _, err := into.Write(s.bytes); err != nil {
		return port.Copy{}, err
	}
	return port.Copy{MediaType: domain.CopyType, Extension: domain.CopyExtension}, nil
}

const videoNote = "notes/https---youtu.be-dQw4w9WgXcQ.url"

// newImportURL is a vault holding one link note, and the store what is
// downloaded for it is kept in.
func newImportURL(t *testing.T, written string, from *site) (ImportURL, *fakeStorage, string) {
	t.Helper()
	shelved := newLibrary()
	shelved.hold(videoNote, domain.KindURL, []byte(written), 1)
	shelf := newShelf()
	paths := []string{}
	return ImportURL{
		Readers: vaults{first.ID: shelved},
		Derived: fakeStores{shelf},
		By:      from,
		Cut: func(_ context.Context, _ domain.Vault, path string) error {
			paths = append(paths, path)
			return nil
		},
	}, shelf, pasted
}

// A file the palette made is named by the address: an address is no filename,
// so the name is what a download replaces once it knows what is there.
const (
	pasted         = "https://youtu.be/dQw4w9WgXcQ"
	aPastedAddress = "[InternetShortcut]\nURL=" + pasted + "\n"
)

const pointsAtAVideo = aPastedAddress

// The words published with a video are written down as the format a player
// opens, under the address they were published at.
func TestTheWordsPublishedWithAVideo(t *testing.T) {
	from := &site{title: "Entropy explained", length: 83_500, cues: []domain.TranscriptCue{
		{Text: "what was said", From: 1500, To: 4200},
		{Text: "what was said next", From: 4200, To: 9100},
	}}
	u, retained, address := newImportURL(t, pointsAtAVideo, from)

	res, err := u.Execute(t.Context(), first, videoNote, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Producer != domain.Captions {
		t.Errorf("the words were downloaded by %q", res.Producer)
	}
	if res.Bytes == 0 {
		t.Error("no words came back")
	}

	hash := text.Fingerprint([]byte(address))
	raw, err := retained.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.Captions, hash))
	if err != nil {
		t.Fatalf("nothing stands under the address: %v", err)
	}
	if got := getWords(t, raw); len(got) != 2 || got[0] != "what was said" {
		t.Errorf("the words read %v", got)
	}
	if _, err := retained.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProducer, domain.Captions, hash)); err != nil {
		t.Errorf("what downloaded the words is not recorded: %v", err)
	}
}

// An address already downloaded is not downloaded again: what a person asked
// for is what stands until they ask for it afresh.
func TestAnAddressAlreadyDownloaded(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []domain.TranscriptCue{{Text: "said", To: 1000}}}
	u, _, _ := newImportURL(t, aPastedAddress, from)

	if _, err := u.Execute(t.Context(), first, videoNote, false); err != nil {
		t.Fatal(err)
	}
	request := len(from.asked)
	if _, err := u.Execute(t.Context(), first, videoNote, false); err != nil {
		t.Fatal(err)
	}
	if len(from.asked) != request {
		t.Errorf("the address was asked again: %v", from.asked)
	}
}

// A video nobody published words for says so, and the address is not asked
// again every time the vault is scanned.
func TestAVideoNobodyPublishedWordsFor(t *testing.T) {
	from := &site{title: "Entropy explained"}
	u, retained, address := newImportURL(t, pointsAtAVideo, from)

	res, err := u.Execute(t.Context(), first, videoNote, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNothing || res.Producer != "" {
		t.Errorf("it came back with %+v", res)
	}
	hash := text.Fingerprint([]byte(address))
	if _, err := retained.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactAnswer, domain.Captions, hash)); err != nil {
		t.Errorf("nothing was written down about it: %v", err)
	}
}

// A page is its prose, without the furniture around it, and it is kept as prose.
func TestThePagePointedAt(t *testing.T) {
	from := &site{title: "Entropy — a page", prose: "A measure of disorder."}
	u, retained, _ := newImportURL(t,
		"[InternetShortcut]\nURL=https://example.com/entropy\n", from)

	res, err := u.Execute(t.Context(), first, videoNote, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Producer != domain.Article {
		t.Errorf("the prose was downloaded by %q", res.Producer)
	}
	hash := text.Fingerprint([]byte("https://example.com/entropy"))
	raw, err := retained.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.Article, hash))
	if err != nil {
		t.Fatalf("nothing stands under the address: %v", err)
	}
	if string(raw) != "A measure of disorder." {
		t.Errorf("the prose reads %q", raw)
	}
	if request := strings.Count(strings.Join(from.asked, " "), "text "); request != 1 {
		t.Errorf("the address was asked for its text %d times: %v", request, from.asked)
	}
}

// Nothing is downloaded for a note that points nowhere.
func TestANoteThatPointsNowhere(t *testing.T) {
	for _, content := range []string{
		"# Entropy\n",
		"[InternetShortcut]\n",
		"[InternetShortcut]\nURL=file:///etc/passwd\n",
	} {
		u, _, _ := newImportURL(t, content, &site{})
		if _, err := u.Execute(t.Context(), first, videoNote, false); err == nil {
			t.Errorf("%q was downloaded for: %v", content, err)
		}
	}
}

// A note still called by the address it points at was named by the paste and by
// nobody. What is there has a name, and the note takes it.
func TestANoteStillCalledByItsAddressConsumesTheTitle(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []domain.TranscriptCue{{Text: "said", To: 1000}}}
	u, _, _ := newImportURL(t, aPastedAddress, from)
	named := []string{}
	u.Names = func(_ context.Context, _ domain.Vault, path, title string) (string, error) {
		named = append(named, path+" → "+title)
		return "notes/Entropy explained.url", nil
	}

	res, err := u.Execute(t.Context(), first, videoNote, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 || named[0] != videoNote+" → Entropy explained" {
		t.Errorf("the note was named %v", named)
	}
	if res.Path != "notes/Entropy explained.url" {
		t.Errorf("the note is filed at %q", res.Path)
	}
}

// A file the person named themselves keeps the name they gave it.
func TestAFilePersonNamedKeepsItsName(t *testing.T) {
	const theirs = "notes/Entropy.url"
	from := &site{title: "Entropy explained", cues: []domain.TranscriptCue{{Text: "said", To: 1000}}}
	u, _, _ := newImportURL(t,
		"[InternetShortcut]\nURL=https://youtu.be/dQw4w9WgXcQ\n", from)
	u.Readers.(vaults)[first.ID].hold(theirs, domain.KindURL, []byte(pointsAtAVideo), 1)
	named := 0
	u.Names = func(context.Context, domain.Vault, string, string) (string, error) {
		named++
		return "", nil
	}

	if _, err := u.Execute(t.Context(), first, theirs, false); err != nil {
		t.Fatal(err)
	}
	if named != 0 {
		t.Errorf("a file the person named was renamed %d times", named)
	}
}

// A copy is kept in the application's own folder, where losing it costs another
// download and the vault stays the person's own writing.
func TestACopyCachedInTheApplicationsFolder(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, retained, address := newImportURL(t, pointsAtAVideo, from)

	got, err := u.Copy(t.Context(), first, videoNote, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.At != "" {
		t.Errorf("the copy landed at %q in the vault, and nothing asked for that", got.At)
	}
	if got.Bytes != int64(len(from.bytes)) {
		t.Errorf("the copy is %d bytes", got.Bytes)
	}
	name := domain.NewCopyName(text.Fingerprint([]byte(address)))
	if _, _, err := retained.OpenArtifact(t.Context(), name); err != nil {
		t.Errorf("nothing in the folder the copy was kept in: %v", err)
	}
}

// A copy already in the application's folder is not downloaded a second
// time, and the file opened to check for it is closed, not leaked.
func TestACopyAlreadyInTheApplicationsFolder(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, retained, _ := newImportURL(t, pointsAtAVideo, from)

	if _, err := u.Copy(t.Context(), first, videoNote, nil); err != nil {
		t.Fatal(err)
	}
	got, err := u.Copy(t.Context(), first, videoNote, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsExisted {
		t.Error("the video was downloaded again over a copy already standing")
	}
	if strings.Count(strings.Join(from.asked, "\n"), "download") != 1 {
		t.Errorf("the site was asked %v", from.asked)
	}
	if retained.closeCount() != retained.opens {
		t.Errorf("checking for an existing copy opened %d files and closed %d", retained.opens, retained.closeCount())
	}
}

// `importing.copies_to_vault` keeps a copy beside the note, as a file the
// person sees in their own folder and plays from where it lies.
func TestACopyCachedAdjacentTheNote(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, retained, address := newImportURL(t, pointsAtAVideo, from)
	target := newLibrary()
	target.hold(videoNote, domain.KindURL, []byte(pointsAtAVideo), 1)
	u.Readers = vaults{first.ID: target}
	u.IsToVault, u.Writers = true, writers{first.ID: target}

	got, err := u.Copy(t.Context(), first, videoNote, nil)
	if err != nil {
		t.Fatal(err)
	}
	besidePath := CopyBeside(videoNote)
	if got.At != besidePath {
		t.Errorf("the copy landed at %q, want it beside the note", got.At)
	}
	if got.Bytes != int64(len(from.bytes)) {
		t.Errorf("the copy is %d bytes", got.Bytes)
	}
	if target.files[besidePath] == nil {
		t.Fatal("nothing stands beside the note")
	}
	if string(target.files[besidePath].raw) != string(from.bytes) {
		t.Errorf("the copy reads %q", target.files[besidePath].raw)
	}
	if _, _, err := retained.OpenArtifact(t.Context(), domain.NewCopyName(text.Fingerprint([]byte(address)))); err == nil {
		t.Error("the copy is in the application's folder too, and the vault is where it was asked for")
	}
}

// A copy already beside the note is not downloaded a second time.
func TestACopyAlreadyAdjacentTheNote(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, _, _ := newImportURL(t, pointsAtAVideo, from)
	target := newLibrary()
	target.hold(videoNote, domain.KindURL, []byte(pointsAtAVideo), 1)
	u.Readers = vaults{first.ID: target}
	u.IsToVault, u.Writers = true, writers{first.ID: target}

	if _, err := u.Copy(t.Context(), first, videoNote, nil); err != nil {
		t.Fatal(err)
	}
	got, err := u.Copy(t.Context(), first, videoNote, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsExisted {
		t.Error("the video was downloaded again over a copy already standing")
	}
	if strings.Count(strings.Join(from.asked, "\n"), "download") != 1 {
		t.Errorf("the site was asked %v", from.asked)
	}
}

// The size gate and the download of one copy run under one operation, so an
// adapter can extract the address once for both.
func TestACopyAsksMetadataAndDownloadBelowOneOperation(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, _, _ := newImportURL(t, pointsAtAVideo, from)

	if _, err := u.Copy(t.Context(), first, videoNote, nil); err != nil {
		t.Fatal(err)
	}
	if len(from.inOperation) != 2 || !from.inOperation[0] || !from.inOperation[1] {
		t.Errorf("metadata and download ran under an operation: %v", from.inOperation)
	}
}
