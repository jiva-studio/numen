package download

import (
	"context"
	"errors"
	"net/url"
	"strings"

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
	page, err := p.web.Fetch(ctx, target)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, err
	}
	defer func() { err = errors.Join(err, page.Close()) }()

	read, err := readability.FromReader(page, address)
	if err != nil {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, ErrNothingDownloaded
	}
	prose := strings.TrimSpace(read.TextContent)
	if prose == "" {
		return Text{Producer: "", Cues: nil, Prose: "", Title: "", Length: 0}, ErrNothingDownloaded
	}
	return Text{Producer: "article", Cues: nil, Prose: prose, Title: strings.TrimSpace(read.Title), Length: 0}, nil
}
