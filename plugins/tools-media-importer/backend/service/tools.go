package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// instructions is the paragraph that tells an agent when to call url_import.
const instructions = "Use url_import to add a web page or a video to the vault by its address. It creates a link " +
	"note named after the address and fetches the text published there, the prose of an article or the " +
	"captions of a video, before it returns; that text can be read and searched from then on. " +
	"Write your own notes about the page as separate notes that link to the link note. Set copy only when " +
	"the person wants the video on this disk: a copy is large and runs in the background, and the result " +
	"names its task."

var errNoVault = errors.New("the call names no vault")

// URLImportInput is the input of the url_import tool.
type URLImportInput struct {
	URL        string `json:"url" jsonschema:"the http or https address to import"`
	Folder     string `json:"folder,omitempty" jsonschema:"the folder of the link note, relative to the vault root; the root by default"`
	ShouldCopy bool   `json:"copy,omitempty" jsonschema:"also copy the video at the address onto this disk; a copy is large, so set it only when the video is wanted offline"`
}

// URLImportOutput is the output of the url_import tool.
type URLImportOutput struct {
	Path       string `json:"path" jsonschema:"the link note created, relative to the vault root"`
	Title      string `json:"title,omitempty" jsonschema:"the title of the page or the video"`
	Producer   string `json:"producer,omitempty" jsonschema:"what fetched the text: captions for the words of a video, article for the prose of a page"`
	Bytes      int    `json:"bytes,omitempty" jsonschema:"the size of the fetched text in bytes"`
	IsNothing  bool   `json:"nothing,omitempty" jsonschema:"the address publishes no text; this is a result, not a failure"`
	Error      string `json:"error,omitempty" jsonschema:"why the text was not fetched; the link note is kept"`
	CopyTaskID string `json:"copy_task_id,omitempty" jsonschema:"the id of the background copy of the video in the list of tasks"`
	CopyError  string `json:"copy_error,omitempty" jsonschema:"why no copy was started"`
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
	registry.SetInstructions(instructions)
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "url_import",
		Title: "Import a URL",
		Description: "Create a link note for a web address and fetch the text published there before returning: " +
			"the captions of a video with their times, or the prose of an article. The text is stored as a text " +
			"version of the note, which library_search and library_read read. The note is named after the " +
			"address and holds only the address. An address that is not http or https is refused. A failed " +
			"fetch is reported in error and the note is kept. With copy set, a video is also copied onto this " +
			"disk in the background, and the result names the task of the copy.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
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
	made, err := t.links.Create(ctx, vault, in.Folder, in.URL)
	if err != nil {
		return URLImportOutput{}, err
	}
	out := URLImportOutput{
		Path: made.Path, Title: "", Producer: "", Bytes: 0, IsNothing: false, Error: "", CopyTaskID: "", CopyError: "",
	}
	if err := t.fetchText(ctx, made, &out); err != nil {
		out.Error = err.Error()
	}
	if in.ShouldCopy {
		t.startCopy(ctx, made, &out)
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
	out.Title, out.Producer, out.IsNothing, out.Bytes = info.Title, info.Producer, info.IsNothing, len(text)
	return nil
}

// startCopy copies the video of the link note in the background where the address is copyable, and
// writes the task of the copy or the reason no copy was started into out.
func (t *Tools) startCopy(ctx context.Context, made link.Link, out *URLImportOutput) {
	if !t.links.IsCopyable(made.Address) {
		out.CopyError = link.ErrNotCopyable.Error()
		return
	}
	t.links.StartCopy(ctx, made)
	out.CopyTaskID = link.GetCopyTaskID(made.Path)
}
