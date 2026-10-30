package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

const maxPageBytes = 8 << 20

const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

const readerName = "go-readability"

type PageProvider struct {
	client *http.Client
}

func NewPages() *PageProvider {
	return &PageProvider{client: &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("this address sends the fetch on and on")
			}
			if err := validateURL(req.URL.String()); err != nil {
				return fmt.Errorf("this address sends the fetch to %s: %w", req.URL.Host, err)
			}
			return nil
		},
	}}
}

func (p *PageProvider) CanHandle(target string) bool { return !isVideoSite(target) }

func (p *PageProvider) GetDownloadModel(string) DownloadModel {
	return DownloadModel{Tool: readerName, Producer: "article"}
}

func (p *PageProvider) GetMetadata(ctx context.Context, target string) (Metadata, error) {
	article, err := p.GetText(ctx, target, PreferredCaptions{})
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{Title: article.Title}, nil
}

func (p *PageProvider) GetText(ctx context.Context, target string, _ PreferredCaptions) (text Text, err error) {
	address, err := url.Parse(target)
	if err != nil {
		return Text{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Text{}, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	answer, err := p.client.Do(req)
	if err != nil {
		return Text{}, err
	}
	defer func() { err = errors.Join(err, answer.Body.Close()) }()
	if answer.StatusCode != http.StatusOK {
		return Text{}, fmt.Errorf("%s answered %s", target, answer.Status)
	}

	read, err := readability.FromReader(io.LimitReader(answer.Body, maxPageBytes), address)
	if err != nil {
		return Text{}, ErrNothingDownloaded
	}
	prose := strings.TrimSpace(read.TextContent)
	if prose == "" {
		return Text{}, ErrNothingDownloaded
	}
	return Text{Producer: "article", Title: strings.TrimSpace(read.Title), Prose: prose}, nil
}

func validateURL(raw string) error {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("not a web address")
	}
	scheme := strings.ToLower(address.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("not a web address")
	}
	host := strings.ToLower(address.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return errors.New("local addresses not allowed")
	}
	if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return errors.New("local addresses not allowed")
		}
	}
	return nil
}
