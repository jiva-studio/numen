package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

const maxPageBytes = 8 << 20

const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

const readerName = "go-readability"

// PageProvider reads the article published at a web page.
type PageProvider struct {
	client *http.Client
}

// NewPages returns the provider of web pages other than video sites.
func NewPages() *PageProvider {
	dialer := new(net.Dialer)
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="net.Dialer fields are captured in closure and not observable without waiting 30s"
	dialer.Timeout = 30 * time.Second
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="net.Dialer fields are captured in closure and not observable without waiting 30s"
	dialer.KeepAlive = 30 * time.Second

	transport := new(http.Transport)
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = dialer.DialContext
	transport.ForceAttemptHTTP2 = true
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ExpectContinueTimeout = 1 * time.Second

	client := new(http.Client)
	client.Transport = transport
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("this address sends the fetch on and on")
		}
		return nil
	}
	return &PageProvider{client: client}
}

// CanHandle reports whether the target is a page other than a video site.
func (p *PageProvider) CanHandle(target string) bool { return !isVideoSite(target) }

// GetDownloadModel returns the reader that produces the article.
func (p *PageProvider) GetDownloadModel(string) Model {
	return Model{Tool: readerName, Version: "", Producer: "article"}
}

// GetMetadata returns the title of the article published at the target.
func (p *PageProvider) GetMetadata(ctx context.Context, target string) (Metadata, error) {
	article, err := p.GetText(ctx, target, PreferredCaptions{Languages: nil, IsAutomatic: false})
	if err != nil {
		return Metadata{Title: "", Bytes: 0, Length: 0, Language: "", Captions: nil, IsAutomatic: nil}, err
	}
	return Metadata{Title: article.Title, Bytes: 0, Length: 0, Language: "", Captions: nil, IsAutomatic: nil}, nil
}

// GetText returns the article published at the target.
func (p *PageProvider) GetText(ctx context.Context, target string, _ PreferredCaptions) (text Text, err error) {
	address, err := url.Parse(target)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	answer, err := p.client.Do(req)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	defer func() { err = errors.Join(err, answer.Body.Close()) }()
	if answer.StatusCode != http.StatusOK {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, fmt.Errorf("%s answered %s", target, answer.Status)
	}

	read, err := readability.FromReader(io.LimitReader(answer.Body, maxPageBytes), address)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, ErrNothingDownloaded
	}
	prose := strings.TrimSpace(read.TextContent)
	if prose == "" {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, ErrNothingDownloaded
	}
	return Text{Producer: "article", Cues: nil, Prose: prose, Title: strings.TrimSpace(read.Title), Length: 0}, nil
}
