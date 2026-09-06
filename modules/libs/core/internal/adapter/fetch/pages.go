package fetch

import (
	"context"
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

// A page is fetched as a person's browser would fetch it, and what it says is
// bytes from a stranger: the parse is somebody else's library, fed a bounded
// read.
type pages struct{ through *http.Client }

func newPages() *pages {
	return &pages{through: &http.Client{Timeout: 30 * time.Second}}
}

// look is what a page calls itself, which is the whole of what a look at one
// needs. It is the same fetch the prose comes out of, and a page is small.
func (p *pages) look(ctx context.Context, at domain.WebAddress) (port.Found, error) {
	article, err := p.prose(ctx, at)
	if err != nil {
		return port.Found{}, err
	}
	return port.Found{Title: article.Title}, nil
}

// prose is the article a page is written around.
func (p *pages) prose(ctx context.Context, at domain.WebAddress) (port.Article, error) {
	address, err := url.Parse(at.URL)
	if err != nil {
		return port.Article{}, err
	}
	asked, err := http.NewRequestWithContext(ctx, http.MethodGet, at.URL, nil)
	if err != nil {
		return port.Article{}, err
	}
	answer, err := p.through.Do(asked)
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
