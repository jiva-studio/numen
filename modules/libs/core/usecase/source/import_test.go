package source

import (
	"context"
	"errors"
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
	asked    []string
	refusing error
}

func (s *site) Fetching() port.FetchModel {
	return port.FetchModel{Tool: "a test", Version: "1"}
}

func (s *site) Look(_ context.Context, at domain.WebAddress) (port.Found, error) {
	s.asked = append(s.asked, "look "+at.URL)
	if s.refusing != nil {
		return port.Found{}, s.refusing
	}
	return port.Found{Title: s.title, Length: s.length}, nil
}

func (s *site) Words(
	_ context.Context, at domain.WebAddress, _ []string,
) ([]transcript.Cue, error) {
	s.asked = append(s.asked, "words "+at.URL)
	if len(s.cues) == 0 {
		return nil, port.ErrNothingFetched
	}
	return s.cues, nil
}

func (s *site) Sound(context.Context, domain.WebAddress, io.Writer) error {
	return port.ErrNothingFetched
}

func (s *site) Copy(context.Context, domain.WebAddress, io.Writer) (port.CopyResult, error) {
	return port.CopyResult{}, port.ErrNothingFetched
}

func (s *site) Prose(_ context.Context, at domain.WebAddress) (port.Article, error) {
	s.asked = append(s.asked, "prose "+at.URL)
	if s.prose == "" {
		return port.Article{}, port.ErrNothingFetched
	}
	return port.Article{Title: s.title, Prose: s.prose}, nil
}

const videoNote = "notes/Entropy.md"

// fetching is a vault holding one link note, and the store what is fetched for
// it is kept in.
func fetching(t *testing.T, written string, from *site) (Import, *shelf, string) {
	t.Helper()
	shelved := newLibrary()
	shelved.hold(videoNote, domain.KindNote, []byte(written), 1)
	kept := newShelf()
	cut := []string{}
	return Import{
		Readers: vaults{first.ID: shelved},
		Derived: kept,
		By:      from,
		Cut: func(_ context.Context, _ domain.Vault, path string) error {
			cut = append(cut, path)
			return nil
		},
	}, kept, "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
}

const pointsAtAVideo = "---\ntype: link\nurl: https://youtu.be/dQw4w9WgXcQ\n---\n\nMine.\n"

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
	u, _, _ := fetching(t, pointsAtAVideo, from)

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

// A page is its prose, without the furniture around it, and it is kept as prose
// rather than as words with times.
func TestThePagePointedAt(t *testing.T) {
	from := &site{title: "Entropy — a page", prose: "A measure of disorder."}
	u, kept, _ := fetching(t,
		"---\ntype: link\nurl: https://example.com/entropy\n---\n\nMine.\n", from)

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
	if strings.Contains(strings.Join(from.asked, " "), "words ") {
		t.Errorf("a page was asked for the words of a video: %v", from.asked)
	}
}

// Nothing is fetched for a note that points nowhere.
func TestANoteThatPointsNowhere(t *testing.T) {
	for _, written := range []string{
		"# Entropy\n",
		"---\ntype: link\n---\n",
		"---\ntype: link\nurl: file:///etc/passwd\n---\n",
	} {
		u, _, _ := fetching(t, written, &site{})
		if _, err := u.Execute(t.Context(), first, videoNote); !errors.Is(err, ErrNotALink) {
			t.Errorf("%q was fetched for: %v", written, err)
		}
	}
}
