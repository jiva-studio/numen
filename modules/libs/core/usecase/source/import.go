package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ErrNotALink is a path holding something other than a note that points
// somewhere. Nothing is fetched for it.
var ErrNotALink = errors.New("this note points nowhere")

// ImportURL fetches what is at the address a link note points at and writes it
// into the vault's own folder.
//
// A person asks for it, once, by pasting the address. What comes back is an
// artifact: no machine here makes it again, and a site may stop publishing it.
// It is named by the address rather than by the note's bytes, so typing in the
// note leaves it where it is and two notes pointing at one video share it.
type ImportURL struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	By      port.Fetcher

	// Languages are the languages published words are preferred in, best
	// first, and Automatic is whether words a machine wrote count where a
	// person published none.
	Languages []string
	Automatic bool

	// Cut brings the note level in the index, so what was fetched is searched
	// with the note as soon as it is written.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	// Again throws away what a run before this one fetched and asks the address
	// afresh. It is how a person asks for a site's words again, and nothing
	// sets it on its own.
	Again bool
}

// ImportURLResult reports what fetching did.
type ImportURLResult struct {
	Path string
	// Title is what the address calls itself, and Length how long a video runs
	// in milliseconds.
	Title  string
	Length int
	// Producer is what fetched the words, and is empty where nothing did.
	Producer string
	// Words is how much text came back, in bytes.
	Words int
	// Nothing is the address publishing none of what was asked for, which is an
	// answer and not a failure.
	Nothing bool
	// Busy is another run holding this address.
	Busy bool
}

// Execute fetches what is at one link note's address.
func (u ImportURL) Execute(ctx context.Context, v domain.Vault, path string) (ImportURLResult, error) {
	res := ImportURLResult{Path: path}
	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	n := markdown.Parse(ref, raw)
	if n.Type != domain.TypeLink {
		return res, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	at, wrong := domain.ReadAddress(n.Frontmatter)
	if len(wrong) > 0 {
		return res, fmt.Errorf("%s: %w: %s", path, ErrNotALink, strings.Join(wrong, "; "))
	}

	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}
	hash := text.Fingerprint([]byte(at.URL))
	from := text.Captions
	if !at.IsVideo() {
		from = text.Article
	}

	// One run to an address. The name is held for as long as the fetch takes,
	// so two notes pointing at one video do not fetch it twice.
	release, err := store.Claim(ctx, text.Partial(from, hash))
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	if u.Again {
		for _, producer := range text.Producers() {
			for _, name := range text.Names(producer, hash) {
				if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return res, err
				}
			}
		}
	}

	// An address already fetched is not fetched again. What a person asked for
	// is what stands, until they ask for it afresh.
	if words, producer, err := text.Fetched(ctx, store, hash); err != nil {
		return res, err
	} else if producer != "" {
		res.Producer, res.Words = producer, len(words)
		return res, nil
	}

	found, err := u.By.Look(ctx, at)
	if err != nil {
		return res, err
	}
	res.Title, res.Length = found.Title, found.Length

	if at.IsVideo() {
		return u.video(ctx, v, ref, at, hash, store, res)
	}
	return u.page(ctx, v, ref, at, hash, store, res)
}

// video writes down the words published with a video, and hands the video to a
// model where nobody published any.
func (u ImportURL) video(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.WebAddress,
	hash string,
	store port.DerivedStore,
	res ImportURLResult,
) (ImportURLResult, error) {
	cues, err := u.By.Words(ctx, at, u.Languages)
	switch {
	case errors.Is(err, port.ErrNothingFetched):
		// Nobody published words for it. That is an answer, and it is written
		// down so the address is not asked again every time the vault is
		// scanned. A copy of the video in the vault is a recording, and what a
		// recording says is heard by the run that hears every other one.
		res.Nothing = true
		return res, store.Write(ctx, text.Answer(text.Captions, hash), []byte(text.Silent+"\n"))
	case err != nil:
		return res, err
	}
	written := transcript.Marshal(cues)
	if err := store.Write(ctx, text.Artifact(text.Captions, hash), written); err != nil {
		return res, err
	}
	if err := u.record(ctx, store, text.Captions, hash, at, res); err != nil {
		return res, err
	}
	res.Producer, res.Words = text.Captions, len(written)
	return res, u.cut(ctx, v, ref.Path)
}

// page writes down the prose of a page.
func (u ImportURL) page(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.WebAddress,
	hash string,
	store port.DerivedStore,
	res ImportURLResult,
) (ImportURLResult, error) {
	article, err := u.By.Prose(ctx, at)
	switch {
	case errors.Is(err, port.ErrNothingFetched):
		res.Nothing = true
		return res, nil
	case err != nil:
		return res, err
	}
	if article.Title != "" {
		res.Title = article.Title
	}
	if err := store.Write(ctx, text.Artifact(text.Article, hash), []byte(article.Prose)); err != nil {
		return res, err
	}
	if err := u.record(ctx, store, text.Article, hash, at, res); err != nil {
		return res, err
	}
	res.Producer, res.Words = text.Article, len(article.Prose)
	return res, u.cut(ctx, v, ref.Path)
}

// record keeps what fetched an address beside what it brought back. Nothing on
// any path that answers a question reads it: it is there so a person can ask
// what produced a text they are reading, and so everything one tool produced
// can be found again.
func (u ImportURL) record(
	ctx context.Context,
	store port.DerivedStore,
	producer, hash string,
	at domain.WebAddress,
	res ImportURLResult,
) error {
	written, err := json.Marshal(struct {
		Address  string `json:"address"`
		Title    string `json:"title,omitempty"`
		Length   int    `json:"length,omitempty"`
		Producer string `json:"producer"`
		Fetcher  string `json:"fetcher"`
	}{
		Address:  at.URL,
		Title:    res.Title,
		Length:   res.Length,
		Producer: producer,
		Fetcher:  u.By.Fetching().Recipe(),
	})
	if err != nil {
		return err
	}
	return store.Write(ctx, text.Beside(producer, hash), written)
}

// cut brings the note level in the index, so the words are searched with it.
func (u ImportURL) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}
