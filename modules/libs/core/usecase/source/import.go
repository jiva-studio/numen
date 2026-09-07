package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ErrNotAURL is a path holding something other than a file naming an address.
// Nothing is fetched for it.
var ErrNotAURL = errors.New("this file holds no web address")

// ImportURL fetches what is at the address a url points at and writes it into
// the vault's own folder.
//
// A person asks for it, once, by pasting the address. What comes back is an
// artifact: no machine here makes it again, and a site may stop publishing it.
// It is named by the address, so renaming the file leaves it where it is and
// two urls on one address share it.
type ImportURL struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	By      port.Fetcher

	// CopyMaxSize is how many bytes a copy may run to. Zero is no limit.
	CopyMaxSize int64

	// ToVault keeps a copy beside the url as a file of the person's own, and
	// Writers is what puts it there. It is `importing.copies_to_vault`, and a
	// run without a writer keeps every copy in the application's own folder.
	ToVault bool
	Writers port.VaultWriters

	// Languages are the languages published words are preferred in, best
	// first, and Automatic is whether words a machine wrote count where a
	// person published none.
	Languages []string
	Automatic bool

	// Cut brings the url level in the index, so what was fetched is searched
	// with it as soon as it is written.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	// Names gives a url the name what is at its address calls itself, and
	// answers where it is filed afterwards. A run given none leaves it called
	// what it was called.
	Names func(ctx context.Context, v domain.Vault, path, title string) (string, error)

	// Again throws away what a run before this one fetched and asks the address
	// afresh. It is how a person asks for a site's words again, and nothing
	// sets it on its own.
	Again bool

	// Progress is told how much of a copy has arrived out of how much the site
	// declared. A run given none is not followed.
	Progress func(done, total int64)
}

// ImportURLResult reports what fetching did.
type ImportURLResult struct {
	Path string
	// Title is what the address calls itself, and Length how long it runs in
	// milliseconds.
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

// Execute fetches what is at one url's address.
func (u ImportURL) Execute(ctx context.Context, v domain.Vault, path string) (ImportURLResult, error) {
	res := ImportURLResult{Path: path}
	at, ref, store, err := u.pointed(ctx, v, path)
	if err != nil {
		return res, err
	}
	hash := text.Fingerprint([]byte(at.URL))
	from := text.Captions
	if !at.IsVideo() {
		from = text.Article
	}

	// One run to an address. The name is held for as long as the fetch takes,
	// so two urls on one address do not fetch it twice.
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

	// A page is one fetch: what it calls itself and the prose it is written
	// around come out of the same read, and a stranger's site is asked once.
	if !at.IsVideo() {
		res, err := u.page(ctx, v, ref, at, hash, store, res)
		if err != nil {
			return res, err
		}
		return u.named(ctx, v, ref, at, res)
	}

	meta, err := u.By.Metadata(ctx, at)
	if err != nil {
		return res, err
	}
	res.Title, res.Length = meta.Title, meta.Length

	res, err = u.video(ctx, v, ref, at, meta, hash, store, res)
	if err != nil {
		return res, err
	}
	return u.named(ctx, v, ref, at, res)
}

// Forget throws away the text fetched from a url's address. The url stands as
// it was, pointing where it points, and the copy fetched for it is untouched.
func (u ImportURL) Forget(ctx context.Context, v domain.Vault, path string) error {
	at, _, store, err := u.pointed(ctx, v, path)
	if err != nil {
		return err
	}
	hash := text.Fingerprint([]byte(at.URL))
	for _, producer := range text.Producers() {
		for _, name := range text.Names(producer, hash) {
			if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}
	return u.cut(ctx, v, path)
}

// CopyResult reports what downloading a copy did.
type CopyResult struct {
	Path string
	// Bytes is how large what is at the address is, and Limit the size a copy
	// may be. Over the limit nothing is fetched.
	Bytes int64
	Limit int64
	// Existed is a copy that already stood, so nothing was fetched.
	Existed bool
	Busy    bool
	// At is where in the vault the copy landed, and nothing where it landed in
	// the application's own folder.
	At string
}

// TooLarge is a copy the limit refused. Nothing was fetched.
func (r CopyResult) TooLarge() bool { return r.Limit > 0 && r.Bytes > r.Limit }

// Copy fetches what is at a url's address so a person plays it from this disk.
//
// It is asked for by hand: a copy takes as much of somebody's disk as what is
// at the address takes, and pasting an address is not asking for one. Where it
// lands is `importing.copies_to_vault`: the application's own folder, where
// losing it costs another fetch, or beside the url as a file of the person's
// own.
func (u ImportURL) Copy(ctx context.Context, v domain.Vault, path string) (CopyResult, error) {
	res := CopyResult{Path: path, Limit: u.CopyMaxSize}
	at, _, store, err := u.pointed(ctx, v, path)
	if err != nil {
		return res, err
	}
	if !at.IsVideo() {
		return res, fmt.Errorf("%s: %w", path, ErrNotAURL)
	}
	hash := text.Fingerprint([]byte(at.URL))
	beside := CopyBeside(path)

	// One run to a copy, held for as long as the fetch takes. The claim is on
	// the store's name whichever disk the copy lands on: it is the address that
	// is being fetched, and two urls on one address are one fetch.
	release, err := store.Claim(ctx, text.Copy(hash))
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	if size, held, err := u.stands(ctx, v, store, hash, beside); err != nil {
		return res, err
	} else if held {
		res.Existed, res.Bytes, res.At = true, size, u.landing(beside)
		return res, nil
	}

	meta, err := u.By.Metadata(ctx, at)
	if err != nil {
		return res, err
	}
	res.Bytes = meta.Bytes
	if u.CopyMaxSize > 0 && meta.Bytes > u.CopyMaxSize {
		return res, nil
	}

	// The copy is written through the store's own rename, so a fetch that
	// stopped leaves nothing anything plays.
	read, write := io.Pipe()
	going := make(chan error, 1)
	go func() {
		_, err := u.By.Download(ctx, at, write)
		_ = write.CloseWithError(err)
		going <- err
	}()
	arriving := capped(read, u.CopyMaxSize)
	if u.Progress != nil {
		u.Progress(0, meta.Bytes)
		arriving = &progressReader{from: arriving, total: meta.Bytes, report: u.Progress}
	}
	size, err := u.takes(ctx, v, store, text.Copy(hash), beside, arriving)
	// The read end is closed before the fetch is waited for: whoever stopped
	// reading unblocks whoever is writing into it.
	_ = read.CloseWithError(err)
	fetching := <-going
	// A site that declared a size it then ran past is known only to be over the
	// limit, which is the size reported.
	if errors.Is(err, errTooLarge) {
		res.Bytes = u.CopyMaxSize + 1
		return res, nil
	}
	if fetching != nil {
		return res, fetching
	}
	if err != nil {
		return res, err
	}
	res.Bytes, res.At = size, u.landing(beside)
	return res, nil
}

// CopyBeside is where a copy kept in the vault stands: beside the url, under
// its own name. A person looking at the folder sees the copy and the url
// together.
func CopyBeside(path string) string {
	return strings.TrimSuffix(path, domain.URLExtension) + text.CopyExtension
}

// landing is where the copy is, as the vault names it, and nothing where it is
// in the application's own folder.
func (u ImportURL) landing(beside string) string {
	if !u.ToVault {
		return ""
	}
	return beside
}

// stands says whether a copy is already here, and how large.
func (u ImportURL) stands(
	ctx context.Context, v domain.Vault, store port.DerivedStore, hash, beside string,
) (int64, bool, error) {
	if u.ToVault {
		reader, err := u.Readers.Open(v)
		if err != nil {
			return 0, false, err
		}
		ref, err := reader.Stat(ctx, beside)
		if errors.Is(err, fs.ErrNotExist) {
			return 0, false, nil
		}
		if err != nil {
			return 0, false, err
		}
		return ref.Size, true, nil
	}
	_, size, err := store.Open(ctx, text.Copy(hash))
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return size, true, nil
}

// takes writes the copy where the settings say it lands, and answers how large
// it came to.
//
// A copy in the vault is the person's file: the application does not replace
// one that is there, and putting the folders above it right is the writer's.
func (u ImportURL) takes(
	ctx context.Context, v domain.Vault, store port.DerivedStore, name, beside string, from io.Reader,
) (int64, error) {
	if !u.ToVault {
		return store.Take(ctx, name, from)
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return 0, err
	}
	counted := &counting{from: from}
	if err := writer.Bring(ctx, beside, counted); err != nil {
		return 0, err
	}
	return counted.read, nil
}

// counting is what went past, so a copy the vault wrote says how large it is
// without being asked for again.
type counting struct {
	from io.Reader
	read int64
}

func (c *counting) Read(into []byte) (int, error) {
	read, err := c.from.Read(into)
	c.read += int64(read)
	return read, err
}

// progressReader reports how much has arrived as it is read. It reports every
// megabyte: finer than that is past what a bar shows, and every report crosses
// to the window.
type progressReader struct {
	from     io.Reader
	total    int64
	report   func(done, total int64)
	done     int64
	reported int64
}

const progressStep = 1 << 20

func (p *progressReader) Read(into []byte) (int, error) {
	read, err := p.from.Read(into)
	p.done += int64(read)
	if p.done-p.reported >= progressStep {
		p.reported = p.done
		p.report(p.done, p.total)
	}
	return read, err
}

// errTooLarge is a copy that ran past the size the settings name.
var errTooLarge = errors.New("over the size a copy may be")

// capped is what is fetched, stopped at the size the settings name. A site that
// declares no size is held to it all the same, and a limit of nothing lets
// everything through.
func capped(from io.Reader, under int64) io.Reader {
	if under <= 0 {
		return from
	}
	return &limited{from: from, left: under + 1}
}

type limited struct {
	from io.Reader
	left int64
}

func (l *limited) Read(into []byte) (int, error) {
	if l.left <= 0 {
		return 0, errTooLarge
	}
	if int64(len(into)) > l.left {
		into = into[:l.left]
	}
	read, err := l.from.Read(into)
	l.left -= int64(read)
	if l.left <= 0 {
		return read, errTooLarge
	}
	return read, err
}

// pointed is one url: where it points, the file itself, and the store what is
// fetched for it is kept in.
func (u ImportURL) pointed(
	ctx context.Context, v domain.Vault, path string,
) (domain.WebAddress, domain.Fingerprint, port.DerivedStore, error) {
	none := domain.WebAddress{}
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
	at, err := domain.ReadURL(raw)
	if err != nil {
		return none, domain.Fingerprint{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return none, domain.Fingerprint{}, nil, err
	}
	return at, ref, store, nil
}

// named gives the url the name what is at the address calls itself.
//
// A url still called by the address it points at was named by the paste and by
// nobody: the person had no name for it yet, and what is there has one. One
// called anything else was named by the person, and that stands.
func (u ImportURL) named(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.WebAddress,
	res ImportURLResult,
) (ImportURLResult, error) {
	if u.Names == nil || res.Title == "" {
		return res, nil
	}
	// A file still called what the paste called it is one nobody has named. Any
	// other name is the person's, and it stands.
	pasted, _, err := domain.Filename(at.URL)
	if err != nil || domain.Basename(ref.Path) != pasted {
		return res, nil
	}
	path, err := u.Names(ctx, v, ref.Path, res.Title)
	if err != nil {
		return res, err
	}
	if path != "" {
		res.Path = path
	}
	return res, nil
}

// video writes down the words published with it, and hands it to a model where
// nobody published any.
func (u ImportURL) video(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.WebAddress,
	meta port.Metadata,
	hash string,
	store port.DerivedStore,
	res ImportURLResult,
) (ImportURLResult, error) {
	cues, err := u.By.Subtitles(ctx, at, language(meta, u.Languages, u.Automatic))
	switch {
	case errors.Is(err, port.ErrNothingFetched):
		// Nobody published words for it. That is an answer, and it is written
		// down so the address is not asked again every time the vault is
		// scanned. A copy in the vault is a recording, and what a recording
		// says is heard by the run that hears every other one.
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

// language is the one the words are asked for in.
//
// What a person published is preferred over what a machine wrote; among those,
// the languages this installation named, and then the language it was spoken
// in. Something translated into thirty languages publishes words in all thirty,
// and what was said in it is one of them.
func language(meta port.Metadata, languages []string, automatic bool) string {
	tracks := meta.Captions
	if len(tracks) == 0 && automatic {
		tracks = meta.Automatic
	}
	if len(tracks) == 0 {
		return ""
	}
	for _, wanted := range append(append([]string(nil), languages...), meta.Language) {
		if one := slices.IndexFunc(tracks, in(wanted)); one >= 0 {
			return tracks[one]
		}
	}
	if one := slices.IndexFunc(tracks, original); one >= 0 {
		return tracks[one]
	}
	return tracks[0]
}

// in says whether a track is in one language. A machine's own is that language
// with a word after it, and its translations of that one are other languages.
func in(language string) func(string) bool {
	return func(track string) bool { return track == language || track == language+"-orig" }
}

// original says whether a track is the language it was spoken in.
func original(track string) bool { return strings.HasSuffix(track, "-orig") }

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
	article, err := u.By.Article(ctx, at)
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
		Fetcher:  u.By.Fetching(at).Recipe(),
	})
	if err != nil {
		return err
	}
	return store.Write(ctx, text.Beside(producer, hash), written)
}

// cut brings the url level in the index, so the words are searched with it.
func (u ImportURL) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}
