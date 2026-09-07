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
)

// mostBytes is the most of a page that is read. A page is an article somebody
// wrote, and something larger is a download wearing a page's clothes.
const mostBytes = 8 << 20

// browser is what a page is asked for as. A great many sites answer a request
// carrying no such name with a refusal.
const browser = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// A page is fetched as a person's browser would fetch it, and what it says is
// bytes from a stranger: the parse is somebody else's library, fed a bounded
// read.
// readerName is what a page's prose is claimed by, which is the library that
// found it.
const readerName = "go-readability"

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

// metadata is what a page calls itself, which is the whole of what is known
// about one before it is read. It is the same fetch the prose comes out of.
func (p *pages) metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	article, err := p.article(ctx, at)
	if err != nil {
		return port.Metadata{}, err
	}
	return port.Metadata{Title: article.Title}, nil
}

// article is the prose a page is written around.
func (p *pages) article(ctx context.Context, at domain.WebAddress) (port.Article, error) {
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
