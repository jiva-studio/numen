package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// mostBytes is the most of a page that is read. A page is an article somebody
// wrote, and something larger is a download wearing a page's clothes.
const mostBytes = 8 << 20

// browser is what a page is asked for as. A great many sites answer a request
// carrying no such name with a refusal.
const browser = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// readerName is what a page's prose is claimed by, which is the library that
// found it.
const readerName = "go-readability"

// pages reaches an address nothing else does, which is every address a browser
// would go to and no tool here knows better. It is fetched as a person's
// browser would fetch it, and what it says is bytes from a stranger: the parse
// is somebody else's library, fed a bounded read.
//
// It runs in this process, so every machine has it.
type pages struct{ through *http.Client }

func newPages() *pages {
	return &pages{through: &http.Client{
		Timeout: 30 * time.Second,
		// A site may send the fetch on, and where it lands is judged as the
		// address was: a redirect onto this machine is refused.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("this address sends the fetch on and on")
			}
			if _, err := domain.ParseWebAddress(req.URL.String()); err != nil {
				return fmt.Errorf("this address sends the fetch to %s: %w", req.URL.Host, err)
			}
			return nil
		},
	}}
}

// Reaches is anything published as a page, which is every address a site does
// not publish as a video. It stands last, and it is what takes whatever the
// strategies before it did not.
func (p *pages) Reaches(at domain.WebAddress) bool { return !at.IsVideo() }

func (p *pages) Fetching(domain.WebAddress) port.FetchModel {
	return port.FetchModel{Tool: readerName}
}

// Subtitles are words with times in them, which a page has none of.
func (p *pages) Subtitles(
	context.Context, domain.WebAddress, string,
) ([]transcript.Cue, error) {
	return nil, port.ErrNothingFetched
}

// Audio is a recording, which a page is not.
func (p *pages) Audio(context.Context, domain.WebAddress, io.Writer) error {
	return port.ErrNothingFetched
}

// Download is a copy a person plays, and a page is read and not played.
func (p *pages) Download(
	context.Context, domain.WebAddress, io.Writer,
) (port.Download, error) {
	return port.Download{}, port.ErrNothingFetched
}

// Metadata is what a page calls itself, which is the whole of what is known
// about one before it is read. It is the same fetch the prose comes out of.
func (p *pages) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	article, err := p.Article(ctx, at)
	if err != nil {
		return port.Metadata{}, err
	}
	return port.Metadata{Title: article.Title}, nil
}

// Article is the prose a page is written around.
func (p *pages) Article(ctx context.Context, at domain.WebAddress) (port.Article, error) {
	address, err := url.Parse(at.URL)
	if err != nil {
		return port.Article{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, at.URL, nil)
	if err != nil {
		return port.Article{}, err
	}
	// A page is asked for the way the browser this person pasted the address
	// out of would ask for it. A site that refuses everything else refuses this
	// too, and says so in its answer.
	req.Header.Set("User-Agent", browser)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	answer, err := p.through.Do(req)
	if err != nil {
		return port.Article{}, err
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return port.Article{}, fmt.Errorf("%s answered %s", at.URL, answer.Status)
	}

	raw, err := io.ReadAll(io.LimitReader(answer.Body, mostBytes))
	if err != nil {
		return port.Article{}, err
	}
	read, err := readability.FromReader(strings.NewReader(string(raw)), address)
	if err != nil {
		return port.Article{}, port.ErrNothingFetched
	}
	prose := strings.TrimSpace(read.TextContent)
	if prose == "" {
		return port.Article{}, port.ErrNothingFetched
	}
	return port.Article{Title: strings.TrimSpace(read.Title), Prose: prose}, nil
}
