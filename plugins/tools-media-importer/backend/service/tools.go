package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// The sentences that tell an agent when to call url_import.
const (
	importerUseSentence      = "Use url_import to add a web page or a video to the vault by its address."
	importerFetchSentence    = "url_import creates a link note named after the address and fetches the text published there, the prose of an article or the captions of a video, before it returns; that text can be read and searched from then on."
	importerOwnNotesSentence = "Write your own notes about the page as separate notes that link to the link note."
	importerDownloadSentence = "Set downloadVideo only when the user wants the video on this disk: the download is large and runs in the background, and the result returns its task id."
)

var errNoVault = errors.New("the call names no vault")

// URLImportInput is the input of the url_import tool.
type URLImportInput struct {
	URL                 string `json:"url" jsonschema:"the http or https address to import"`
	Folder              string `json:"folder,omitempty" jsonschema:"the folder of the link note, relative to the vault root; the root by default"`
	ShouldDownloadVideo bool   `json:"downloadVideo,omitempty" jsonschema:"also download the video at the address to this disk; the download is large, so set it only when the video is needed offline"`
}

// URLImportOutput is the output of the url_import tool.
type URLImportOutput struct {
	Path           string `json:"path" jsonschema:"the link note created, relative to the vault root"`
	Title          string `json:"title,omitempty" jsonschema:"the title of the page or the video"`
	Producer       string `json:"producer,omitempty" jsonschema:"what fetched the text: captions for the words of a video, article for the prose of a page"`
	Bytes          int    `json:"bytes,omitempty" jsonschema:"the size of the fetched text in bytes"`
	HasText        bool   `json:"hasText" jsonschema:"true when text was fetched; false when the address publishes no text or the fetch failed"`
	Error          string `json:"error,omitempty" jsonschema:"why the text was not fetched; the link note is kept"`
	DownloadTaskID string `json:"downloadTaskId,omitempty" jsonschema:"the id of the background download of the video in the list of tasks"`
	DownloadError  string `json:"downloadError,omitempty" jsonschema:"the reason no download was started"`
}

// Tools serves url_import.
type Tools struct {
	links *link.Links
}

// NewTools is the tools over the links.
func NewTools(links *link.Links) *Tools {
	return &Tools{links: links}
}

// Registry registers and returns the import tool.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "url_import",
		Title: "Import a URL",
		Description: "Create a link note for a web address and fetch the text published there before returning: " +
			"the captions of a video with their times, or the prose of an article. The text is stored as a text " +
			"version of the note, which library_search and library_read read. The note is named after the " +
			"address and holds only the address; the link note of the same address already in the folder is " +
			"kept, and its text is fetched again. An address that is not http or https is refused. A failed " +
			"fetch is reported in error and the note is kept. With downloadVideo set, the video is also " +
			"downloaded to this disk in the background, and the result returns the task id of the download.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{importerUseSentence, importerFetchSentence, importerOwnNotesSentence, importerDownloadSentence},
	}, t.importURL)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func (t *Tools) importURL(ctx context.Context, in URLImportInput) (URLImportOutput, error) {
	vault := domain.VaultID(tools.GetVault(ctx))
	if vault == "" {
		return URLImportOutput{}, errNoVault
	}
	made, err := t.links.FindOrCreate(ctx, vault, in.Folder, in.URL)
	if err != nil {
		return URLImportOutput{}, err
	}
	out := URLImportOutput{
		Path: made.Path, Title: "", Producer: "", Bytes: 0, HasText: false, Error: "", DownloadTaskID: "", DownloadError: "",
	}
	if err := t.fetchText(ctx, made, &out); err != nil {
		out.Error = err.Error()
	}
	if in.ShouldDownloadVideo {
		t.startDownload(ctx, made, &out)
	}
	return out, nil
}

// fetchText fetches the text of the link note and writes what was fetched into out.
func (t *Tools) fetchText(ctx context.Context, made link.Link, out *URLImportOutput) error {
	if err := t.links.FetchText(ctx, made, true); err != nil {
		return err
	}
	info, err := t.links.GetTextInfo(ctx, made)
	if err != nil {
		return err
	}
	text, err := t.links.ReadArticle(ctx, made)
	if err != nil {
		return err
	}
	out.Title, out.Producer, out.HasText, out.Bytes = info.Title, info.Producer, !info.IsNothing, len(text)
	return nil
}

// startDownload downloads the video of the link note in the background where the address is
// copyable, and writes the task id of the download or the reason none was started into out.
func (t *Tools) startDownload(ctx context.Context, made link.Link, out *URLImportOutput) {
	if !t.links.IsCopyable(made.Address) {
		out.DownloadError = link.ErrNotCopyable.Error()
		return
	}
	t.links.StartCopy(ctx, made)
	out.DownloadTaskID = link.GetCopyTaskID(made.Path)
}
