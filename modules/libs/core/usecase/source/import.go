package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNotAURL is a path holding something other than a file naming an address.
// Nothing is downloaded for it.
var ErrNotAURL = errors.New("this file holds no web address")

// ImportURL downloads what is at the address a url points at and writes it into
// the vault's own folder.
//
// A person asks for it, once, by pasting the address. What comes back is an
// artifact: no machine here makes it again, and a site may stop publishing it.
// It is named by the address, so renaming the file leaves it where it is and
// two urls on one address share it.
type ImportURL struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	By      port.Downloader

	// CopyMaxSize is how many bytes a copy may run to. Zero is no limit.
	CopyMaxSize int64

	// IsToVault keeps a copy beside the url as a file of the person's own, and
	// Writers is what puts it there. It is `importing.copies_to_vault`, and a
	// run without a writer keeps every copy in the application's own folder.
	IsToVault bool
	Writers   port.VaultWriters

	// Languages are the languages published words are preferred in, best
	// first, and Automatic is whether words a machine wrote count where a
	// person published none.
	Languages   []string
	IsAutomatic bool

	// Cut brings the url level in the index, so what was downloaded is searched
	// with it as soon as it is written.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	// Names gives a url the name what is at its address calls itself, and
	// answers where it is filed afterwards. A run given none leaves it called
	// what it was called.
	Names func(ctx context.Context, v domain.Vault, path, title string) (string, error)

	// IsRepeat throws away what a run before this one downloaded and asks the address
	// afresh. It is how a person asks for a site's words again, and nothing
	// sets it on its own.
	IsRepeat bool

	// Progress is told how much of a copy has arrived out of how much the site
	// declared. A run given none is not followed.
	Progress func(done, total int64)
}

// ErrBeingDownloaded is another run holding this address. Nothing was done, and
// asking again once that run is over is the whole of what is left to do.
var ErrBeingDownloaded = errors.New("another run is downloading this address")

// ImportURLResult reports what the download did.
type ImportURLResult struct {
	// Path is where the url is filed now: what came back says what the file is
	// called, so a run may leave it somewhere else than it found it.
	Path string
	// Producer is what downloaded the text, and is empty where nothing did.
	Producer string
	// Bytes is how much text came back.
	Bytes int
	// IsNothing is the address publishing none of what was asked for, which is an
	// answer and not a failure.
	IsNothing bool
}

// Execute downloads what is at one url's address.
func (u ImportURL) Execute(ctx context.Context, v domain.Vault, path string) (ImportURLResult, error) {
	res := ImportURLResult{Path: path}
	at, ref, store, err := u.readURLFile(ctx, v, path)
	if err != nil {
		return res, err
	}
	hash := text.Fingerprint([]byte(string(at)))

	// One run to an address. The name is held for as long as the download takes,
	// and two urls on one address are one download.
	release, err := store.Claim(ctx, text.Partial(u.By.GetDownloadModel(at).Producer, hash))
	if errors.Is(err, port.ErrClaimed) {
		return res, ErrBeingDownloaded
	}
	if err != nil {
		return res, err
	}
	defer release()

	if u.IsRepeat {
		if err := dropDownloads(ctx, store, hash); err != nil {
			return res, err
		}
	}

	// An address already downloaded is not downloaded again. What a person asked
	// for is what stands, until they ask for it afresh.
	if words, producer, err := text.ReadDownloaded(ctx, store, hash); err != nil {
		return res, err
	} else if producer != "" {
		res.Producer, res.Bytes = producer, len(words)
		return res, nil
	}

	said, err := u.By.Text(ctx, at, port.PreferredCaptions{
		Languages: u.Languages, IsAutomatic: u.IsAutomatic,
	})
	switch {
	case errors.Is(err, port.ErrNothingDownloaded):
		// The address publishes none of what was asked for. That is an answer,
		// and it is written down so the address is not asked again every time
		// the vault is scanned.
		res.IsNothing = true
		return res, u.silent(ctx, at, store, hash)
	case err != nil:
		return res, err
	}

	res.Producer, res.Bytes, err = u.writeDownload(ctx, store, hash, at, said)
	if err != nil {
		return res, err
	}
	if err := u.cut(ctx, v, ref.Path); err != nil {
		return res, err
	}
	return u.renameURL(ctx, v, ref, at, said.Title, res)
}

// writeDownload writes down what came back, under the name of whatever
// downloaded it, and answers how much of it there was.
func (u ImportURL) writeDownload(
	ctx context.Context,
	store port.DerivedStore,
	hash string,
	at domain.URL,
	said port.Text,
) (producer string, bytes int, err error) {
	written := []byte(said.Prose)
	if len(said.Cues) > 0 {
		written = transcript.Marshal(said.Cues)
	}
	if err := store.Write(ctx, text.Artifact(said.Producer, hash), written); err != nil {
		return "", 0, err
	}
	if err := u.record(ctx, store, said.Producer, hash, at, said); err != nil {
		return "", 0, err
	}
	return said.Producer, len(written), nil
}

// silent writes down that the address published none of what was asked for, so
// the address is not asked again every time the vault is scanned.
func (u ImportURL) silent(
	ctx context.Context, at domain.URL, store port.DerivedStore, hash string,
) error {
	return store.Write(ctx, text.Answer(u.By.GetDownloadModel(at).Producer, hash), []byte(text.Silent+"\n"))
}

// readURLFile is one url: where it points, the file itself, and the store what
// is downloaded for it is kept in.
func (u ImportURL) readURLFile(
	ctx context.Context, v domain.Vault, path string,
) (domain.URL, domain.Fingerprint, port.DerivedStore, error) {
	none := domain.URL("")
	reader, err := u.Readers.Open(v)
	if err != nil {
		return none, domain.Fingerprint{}, nil, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return none, domain.Fingerprint{}, nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if ref.Kind != domain.KindURL {
		return none, domain.Fingerprint{}, nil, fmt.Errorf("%s: %w", path, ErrNotAURL)
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return none, domain.Fingerprint{}, nil, fmt.Errorf("read %s: %w", path, err)
	}
	at, err := urlfile.Read(raw)
	if err != nil {
		return none, domain.Fingerprint{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return none, domain.Fingerprint{}, nil, err
	}
	return at, ref, store, nil
}

// renameURL gives the url the name what is at the address calls itself.
//
// A url still called by the address it points at was named by the paste and by
// nobody: the person had no name for it yet, and what is there has one. One
// called anything else was named by the person, and that stands.
func (u ImportURL) renameURL(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.URL,
	title string,
	res ImportURLResult,
) (ImportURLResult, error) {
	if u.Names == nil || title == "" {
		return res, nil
	}
	// A file still called what the paste called it is one nobody has named. Any
	// other name is the person's, and it stands; so does one whose address no
	// filename can be made from, which nothing could have named it by.
	pasted, _, err := domain.Filename(string(at))
	if err != nil || domain.Basename(ref.Path) != pasted {
		return res, nil //nolint:nilerr // an unnameable address is a name that stands
	}
	path, err := u.Names(ctx, v, ref.Path, title)
	if err != nil {
		return res, err
	}
	if path != "" {
		res.Path = path
	}
	return res, nil
}

// record keeps what downloaded an address beside what it brought back. Nothing on
// any path that answers a question reads it: it is there so a person can ask
// what produced a text they are reading, and so everything one tool produced
// can be found again.
func (u ImportURL) record(
	ctx context.Context,
	store port.DerivedStore,
	producer, hash string,
	at domain.URL,
	said port.Text,
) error {
	written, err := json.Marshal(struct {
		Address    string `json:"address"`
		Title      string `json:"title,omitempty"`
		Length     int    `json:"length,omitempty"`
		Producer   string `json:"producer"`
		Downloader string `json:"downloader"`
	}{
		Address:    string(at),
		Title:      said.Title,
		Length:     said.Length,
		Producer:   producer,
		Downloader: u.By.GetDownloadModel(at).Recipe(),
	})
	if err != nil {
		return err
	}
	return store.Write(ctx, text.GetProducerFile(producer, hash), written)
}

// cut brings the url level in the index, so the words are searched with it.
func (u ImportURL) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}
