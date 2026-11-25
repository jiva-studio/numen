package file_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// provider is the file provider plugin, at the edge of the core: it records what it is asked and answers
// with a failure set by the test.
type provider struct {
	calls []string
	err   error
}

func (p *provider) FetchFile(_ context.Context, v domain.Vault, path string) error {
	p.calls = append(p.calls, "fetch "+string(v.ID)+" "+path)
	return p.err
}

func (p *provider) DropDownload(_ context.Context, v domain.Vault, path string) error {
	p.calls = append(p.calls, "drop "+string(v.ID)+" "+path)
	return p.err
}

func newDownloadsHandler(shown window, p *provider) *filewindow.Handler {
	return filewindow.NewHandler(nil, nil, filewindow.NewDownloads(shown, file.NewFetchFile(p), file.NewDropDownload(p)))
}

func TestFetchingAFileAsksTheProviderForTheFileOfTheShownVault(t *testing.T) {
	p := &provider{}
	handler := newDownloadsHandler(window{vault: domain.Vault{ID: "v-1"}}, p)

	_, err := handler.FetchFile(t.Context(), connect.NewRequest(&v1.FetchFileRequest{Path: "Books/Guide.pdf"}))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0] != "fetch v-1 Books/Guide.pdf" {
		t.Errorf("the provider was asked %q", p.calls)
	}
}

func TestDroppingADownloadAsksTheProviderForTheFileOfTheShownVault(t *testing.T) {
	p := &provider{}
	handler := newDownloadsHandler(window{vault: domain.Vault{ID: "v-1"}}, p)

	_, err := handler.DropDownload(t.Context(), connect.NewRequest(&v1.DropDownloadRequest{Path: "Books/Guide.pdf"}))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0] != "drop v-1 Books/Guide.pdf" {
		t.Errorf("the provider was asked %q", p.calls)
	}
}

func TestADownloadRefusedByTheProviderKeepsItsCodeAndMessage(t *testing.T) {
	refusal := connect.NewError(connect.CodeFailedPrecondition, errors.New("The file has changes that are not sent"))
	handler := newDownloadsHandler(window{vault: domain.Vault{ID: "v-1"}}, &provider{err: refusal})

	_, err := handler.DropDownload(t.Context(), connect.NewRequest(&v1.DropDownloadRequest{Path: "a.pdf"}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition || failure.Message() != "The file has changes that are not sent" {
		t.Errorf("the call failed with %v", err)
	}
}

func TestADownloadWithoutAProviderPluginFailsAsAPreconditionThatSaysSo(t *testing.T) {
	missing := &plugindomain.UnavailableError{Capability: "files.provider", Reason: plugindomain.NotInstalled}
	handler := newDownloadsHandler(window{vault: domain.Vault{ID: "v-1"}}, &provider{err: missing})

	_, err := handler.FetchFile(t.Context(), connect.NewRequest(&v1.FetchFileRequest{Path: "a.pdf"}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("the call failed with %v", err)
	}
}

func TestADownloadInAWindowWithoutAVaultFails(t *testing.T) {
	p := &provider{}
	handler := newDownloadsHandler(window{}, p)

	_, err := handler.FetchFile(t.Context(), connect.NewRequest(&v1.FetchFileRequest{Path: "a.pdf"}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition || len(p.calls) != 0 {
		t.Errorf("the call failed with %v and asked the provider %q", err, p.calls)
	}
}
