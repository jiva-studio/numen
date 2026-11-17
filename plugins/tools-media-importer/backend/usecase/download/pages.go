package download

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"

	readability "github.com/go-shiori/go-readability"
)

const readerName = "go-readability"

// PageProvider reads the article published at a web page.
type PageProvider struct {
	web Web
}

// NewPages returns the provider of web pages other than video sites, which fetches them through web.
func NewPages(web Web) *PageProvider {
	return &PageProvider{web: web}
}

// CanHandle reports whether the target is a page other than a video site.
func (p *PageProvider) CanHandle(target string) bool { return !isVideoSite(target) }

// GetDownloadModel returns the reader that produces the article.
func (p *PageProvider) GetDownloadModel(string) fetch.Model {
	return fetch.Model{Tool: readerName, Version: "", Producer: "article"}
}

// GetMetadata returns the title of the article published at the target.
func (p *PageProvider) GetMetadata(ctx context.Context, target string) (fetch.Metadata, error) {
	article, err := p.GetText(ctx, target, fetch.PreferredCaptions{Languages: nil, IsAutomatic: false})
	if err != nil {
		return fetch.Metadata{Title: "", Bytes: 0, Length: 0, Language: "", Captions: nil, IsAutomatic: nil}, err
	}
	return fetch.Metadata{Title: article.Title, Bytes: 0, Length: 0, Language: "", Captions: nil, IsAutomatic: nil}, nil
}

// GetText returns the article published at the target.
func (p *PageProvider) GetText(ctx context.Context, target string, _ fetch.PreferredCaptions) (text fetch.Text, err error) {
	address, err := url.Parse(target)
	if err != nil {
		return fetch.Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	page, err := p.web.Fetch(ctx, target)
	if err != nil {
		return fetch.Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	defer func() { err = errors.Join(err, page.Close()) }()

	read, err := readability.FromReader(page, address)
	if err != nil {
		return fetch.Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, fetch.ErrNothingDownloaded
	}
	prose := strings.TrimSpace(read.TextContent)
	if prose == "" {
		return fetch.Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, fetch.ErrNothingDownloaded
	}
	return fetch.Text{Producer: "article", Cues: nil, Prose: prose, Title: strings.TrimSpace(read.Title), Length: 0}, nil
}
