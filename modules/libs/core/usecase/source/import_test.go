package source

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A site that answers with what a test puts in it, and counts what it was asked
// for: nothing here reaches a network.
type site struct {
	title    string
	length   int
	cues     []transcript.Cue
	prose    string
	bytes    []byte
	asked    []string
	refusing error
}

func (s *site) Fetching(_ domain.WebAddress) port.FetchModel {
	return port.FetchModel{Tool: "a test", Version: "1"}
}

func (s *site) Metadata(_ context.Context, at domain.WebAddress) (port.Metadata, error) {
	s.asked = append(s.asked, "metadata "+at.URL)
	if s.refusing != nil {
		return port.Metadata{}, s.refusing
	}
	return port.Metadata{Title: s.title, Length: s.length}, nil
}

func (s *site) Subtitles(
	_ context.Context, at domain.WebAddress, _ string,
) ([]transcript.Cue, error) {
	s.asked = append(s.asked, "subtitles "+at.URL)
	if len(s.cues) == 0 {
		return nil, port.ErrNothingFetched
	}
	return s.cues, nil
}

func (s *site) Audio(context.Context, domain.WebAddress, io.Writer) error {
	return port.ErrNothingFetched
}

func (s *site) Download(_ context.Context, at domain.WebAddress, into io.Writer) (port.Download, error) {
	s.asked = append(s.asked, "download "+at.URL)
	if len(s.bytes) == 0 {
		return port.Download{}, port.ErrNothingFetched
	}
	if _, err := into.Write(s.bytes); err != nil {
		return port.Download{}, err
	}
	return port.Download{MediaType: text.CopyType, Extension: text.CopyExtension}, nil
}

func (s *site) Article(_ context.Context, at domain.WebAddress) (port.Article, error) {
	s.asked = append(s.asked, "article "+at.URL)
	if s.prose == "" {
		return port.Article{}, port.ErrNothingFetched
	}
	return port.Article{Title: s.title, Prose: s.prose}, nil
}

const videoNote = "notes/https---www.youtube.com-watch-v=dQw4w9WgXcQ.url"

// fetching is a vault holding one link note, and the store what is fetched for
// it is kept in.
func fetching(t *testing.T, written string, from *site) (ImportURL, *shelf, string) {
	t.Helper()
	shelved := newLibrary()
	shelved.hold(videoNote, domain.KindURL, []byte(written), 1)
	kept := newShelf()
	cut := []string{}
	return ImportURL{
		Readers: vaults{first.ID: shelved},
		Derived: shelves{kept},
		By:      from,
		Cut: func(_ context.Context, _ domain.Vault, path string) error {
			cut = append(cut, path)
			return nil
		},
	}, kept, "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
}

// A file the palette made is named by the address: an address is no filename,
// so the name is what a fetch replaces once it knows what is there.
const aPastedAddress = "[InternetShortcut]\nURL=https://youtu.be/dQw4w9WgXcQ\n"

const pointsAtAVideo = aPastedAddress

// The words published with a video are written down as the format a player
// opens, under the address they were published at.
func TestTheWordsPublishedWithAVideo(t *testing.T) {
	from := &site{title: "Entropy explained", length: 83_500, cues: []transcript.Cue{
		{Text: "what was said", From: 1500, To: 4200},
		{Text: "what was said next", From: 4200, To: 9100},
	}}
	u, kept, address := fetching(t, pointsAtAVideo, from)

	res, err := u.Execute(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	if res.Producer != text.Captions {
		t.Errorf("the words were fetched by %q", res.Producer)
	}
	if res.Title != "Entropy explained" || res.Length != 83_500 {
		t.Errorf("it is called %q and runs %d ms", res.Title, res.Length)
	}

	hash := text.Fingerprint([]byte(address))
	raw, err := kept.Read(t.Context(), text.Artifact(text.Captions, hash))
	if err != nil {
		t.Fatalf("nothing stands under the address: %v", err)
	}
	if got := spoken(t, raw); len(got) != 2 || got[0] != "what was said" {
		t.Errorf("the words read %v", got)
	}
	if _, err := kept.Read(t.Context(), text.Beside(text.Captions, hash)); err != nil {
		t.Errorf("what fetched the words is not recorded: %v", err)
	}
}

// An address already fetched is not fetched again: what a person asked for is
// what stands until they ask for it afresh.
func TestAnAddressAlreadyFetched(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []transcript.Cue{{Text: "said", To: 1000}}}
	u, _, _ := fetching(t, aPastedAddress, from)

	if _, err := u.Execute(t.Context(), first, videoNote); err != nil {
		t.Fatal(err)
	}
	asked := len(from.asked)
	if _, err := u.Execute(t.Context(), first, videoNote); err != nil {
		t.Fatal(err)
	}
	if len(from.asked) != asked {
		t.Errorf("the address was asked again: %v", from.asked)
	}
}

// A video nobody published words for says so, and the address is not asked
// again every time the vault is scanned.
func TestAVideoNobodyPublishedWordsFor(t *testing.T) {
	from := &site{title: "Entropy explained"}
	u, kept, address := fetching(t, pointsAtAVideo, from)

	res, err := u.Execute(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Nothing || res.Producer != "" {
		t.Errorf("it came back with %+v", res)
	}
	hash := text.Fingerprint([]byte(address))
	if _, err := kept.Read(t.Context(), text.Answer(text.Captions, hash)); err != nil {
		t.Errorf("nothing was written down about it: %v", err)
	}
}

// A page is its prose, without the furniture around it, and it is kept as prose.
func TestThePagePointedAt(t *testing.T) {
	from := &site{title: "Entropy — a page", prose: "A measure of disorder."}
	u, kept, _ := fetching(t,
		"[InternetShortcut]\nURL=https://example.com/entropy\n", from)

	res, err := u.Execute(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	if res.Producer != text.Article {
		t.Errorf("the prose was fetched by %q", res.Producer)
	}
	hash := text.Fingerprint([]byte("https://example.com/entropy"))
	raw, err := kept.Read(t.Context(), text.Artifact(text.Article, hash))
	if err != nil {
		t.Fatalf("nothing stands under the address: %v", err)
	}
	if string(raw) != "A measure of disorder." {
		t.Errorf("the prose reads %q", raw)
	}
	if strings.Contains(strings.Join(from.asked, " "), "subtitles ") {
		t.Errorf("a page was asked for the words of a video: %v", from.asked)
	}
}

// Nothing is fetched for a note that points nowhere.
func TestANoteThatPointsNowhere(t *testing.T) {
	for _, written := range []string{
		"# Entropy\n",
		"[InternetShortcut]\n",
		"[InternetShortcut]\nURL=file:///etc/passwd\n",
	} {
		u, _, _ := fetching(t, written, &site{})
		if _, err := u.Execute(t.Context(), first, videoNote); err == nil {
			t.Errorf("%q was fetched for: %v", written, err)
		}
	}
}

// One language is asked for and not a list. A site that publishes a machine
// translation into every language it knows answers a request for the lot by
// refusing it.
func TestWhichLanguageTheWordsAreAskedFor(t *testing.T) {
	for _, one := range []struct {
		found     port.Metadata
		languages []string
		automatic bool
		want      string
	}{
		{port.Metadata{Captions: []string{"de", "en", "ru"}}, []string{"ru", "en"}, true, "ru"},
		{port.Metadata{Captions: []string{"de", "en"}}, []string{"ru"}, true, "de"},
		{port.Metadata{Automatic: []string{"ab", "en-orig", "zu"}}, nil, true, "en-orig"},
		{port.Metadata{Automatic: []string{"ab", "en-orig"}}, []string{"en"}, true, "en-orig"},
		{port.Metadata{Automatic: []string{"ab", "en-orig"}}, nil, false, ""},
		{port.Metadata{Captions: []string{"en"}, Automatic: []string{"ru-orig"}}, nil, true, "en"},
		{port.Metadata{}, []string{"en"}, true, ""},
	} {
		if got := language(one.found, one.languages, one.automatic); got != one.want {
			t.Errorf("%+v with %v: asked for %q, want %q",
				one.found, one.languages, got, one.want)
		}
	}
}

// A video somebody translated into thirty languages publishes words in all
// thirty, and what was said in it is one of them.
func TestAVideoTranslatedIntoManyLanguages(t *testing.T) {
	meta := port.Metadata{
		Language: "en",
		Captions: []string{"ar", "de", "en", "es", "zh"},
	}
	if got := language(meta, nil, true); got != "en" {
		t.Errorf("asked for %q, want the language it was spoken in", got)
	}
	if got := language(meta, []string{"de"}, true); got != "de" {
		t.Errorf("asked for %q, want the language this installation named", got)
	}
}

// A note still called by the address it points at was named by the paste and by
// nobody. What is there has a name, and the note takes it.
func TestANoteStillCalledByItsAddressTakesTheTitle(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []transcript.Cue{{Text: "said", To: 1000}}}
	u, _, _ := fetching(t, aPastedAddress, from)
	named := []string{}
	u.Names = func(_ context.Context, _ domain.Vault, path, title string) (string, error) {
		named = append(named, path+" → "+title)
		return "notes/Entropy explained.url", nil
	}

	res, err := u.Execute(t.Context(), first, videoNote)
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
	from := &site{title: "Entropy explained", cues: []transcript.Cue{{Text: "said", To: 1000}}}
	u, _, _ := fetching(t,
		"[InternetShortcut]\nURL=https://youtu.be/dQw4w9WgXcQ\n", from)
	u.Readers.(vaults)[first.ID].hold(theirs, domain.KindURL, []byte(pointsAtAVideo), 1)
	named := 0
	u.Names = func(context.Context, domain.Vault, string, string) (string, error) {
		named++
		return "", nil
	}

	if _, err := u.Execute(t.Context(), first, theirs); err != nil {
		t.Fatal(err)
	}
	if named != 0 {
		t.Errorf("a file the person named was renamed %d times", named)
	}
}

// A copy is kept in the application's own folder, where losing it costs another
// fetch and the vault stays the person's own writing.
func TestACopyKeptInTheApplicationsFolder(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, kept, address := fetching(t, pointsAtAVideo, from)

	got, err := u.Copy(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	if got.At != "" {
		t.Errorf("the copy landed at %q in the vault, and nothing asked for that", got.At)
	}
	if got.Bytes != int64(len(from.bytes)) {
		t.Errorf("the copy is %d bytes", got.Bytes)
	}
	name := text.Copy(text.Fingerprint([]byte(address)))
	if _, _, err := kept.Open(t.Context(), name); err != nil {
		t.Errorf("nothing in the folder the copy was kept in: %v", err)
	}
}

// `importing.copies_to_vault` keeps a copy beside the note, as a file the
// person sees in their own folder and plays from where it lies.
func TestACopyKeptBesideTheNote(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, kept, address := fetching(t, pointsAtAVideo, from)
	held := newLibrary()
	held.hold(videoNote, domain.KindURL, []byte(pointsAtAVideo), 1)
	u.Readers = vaults{first.ID: held}
	u.ToVault, u.Writers = true, writers{first.ID: held}

	got, err := u.Copy(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	beside := CopyBeside(videoNote)
	if got.At != beside {
		t.Errorf("the copy landed at %q, want it beside the note", got.At)
	}
	if got.Bytes != int64(len(from.bytes)) {
		t.Errorf("the copy is %d bytes", got.Bytes)
	}
	if held.files[beside] == nil {
		t.Fatal("nothing stands beside the note")
	}
	if string(held.files[beside].raw) != string(from.bytes) {
		t.Errorf("the copy reads %q", held.files[beside].raw)
	}
	if _, _, err := kept.Open(t.Context(), text.Copy(text.Fingerprint([]byte(address)))); err == nil {
		t.Error("the copy is in the application's folder too, and the vault is where it was asked for")
	}
}

// A copy already beside the note is not fetched a second time.
func TestACopyAlreadyBesideTheNote(t *testing.T) {
	from := &site{bytes: []byte("the bytes of a video")}
	u, _, _ := fetching(t, pointsAtAVideo, from)
	held := newLibrary()
	held.hold(videoNote, domain.KindURL, []byte(pointsAtAVideo), 1)
	u.Readers = vaults{first.ID: held}
	u.ToVault, u.Writers = true, writers{first.ID: held}

	if _, err := u.Copy(t.Context(), first, videoNote); err != nil {
		t.Fatal(err)
	}
	got, err := u.Copy(t.Context(), first, videoNote)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Existed {
		t.Error("the video was fetched again over a copy already standing")
	}
	if strings.Count(strings.Join(from.asked, "\n"), "download") != 1 {
		t.Errorf("the site was asked %v", from.asked)
	}
}
