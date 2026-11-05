package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

var errNoVault = errors.New("the call names no vault")

// Tools provides MCP tools for AI agents interacting with the knowledge vault.
type Tools struct {
	// host provides access to every vault registered with the host.
	host host.Vault
	// vault is the vault of the call being served.
	vault host.Vault
	// texts and search are the text versions and the search index. On the tools they are those of
	// every vault, and nil until WithLibrary; on the tools of a call they are those of its vault.
	texts  host.Texts
	search host.Search
}

// New creates a Tools instance.
func New(vault host.Vault) *Tools {
	return &Tools{host: vault, vault: nil, texts: nil, search: nil}
}

// inVault runs the tool on the vault the call names. A call that names none is refused.
func inVault[In, Out any](t *Tools, run func(*Tools, context.Context, In) (Out, error)) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, in In) (Out, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			var none Out
			return none, errNoVault
		}
		call := &Tools{host: t.host, vault: t.host.With(domain.VaultID(id)), texts: nil, search: nil}
		if t.texts != nil {
			call.texts = t.texts.With(domain.VaultID(id))
		}
		if t.search != nil {
			call.search = t.search.With(domain.VaultID(id))
		}
		return run(call, ctx, in)
	}
}

// Registry registers and returns all vault tools.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	for _, add := range []func(*tools.Registry) error{
		t.addNoteRead,
		t.addNoteWrite,
		t.addNoteHeadings,
		t.addNoteSearch,
		t.addNoteList,
		t.addNoteTitles,
		t.addNoteResolve,
		t.addLibrarySearch,
		t.addLibraryRead,
		t.addLinkList,
		t.addVaultGet,
		t.addVaultList,
		t.addVaultProblems,
		t.addFileRead,
		t.addFileCreate,
	} {
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func formatFingerprint(ref domain.FileStat) string {
	var nanos int64
	if !ref.ModTime.IsZero() {
		nanos = ref.ModTime.UnixNano()
	}
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(nanos, 10)
}

func parseFingerprint(s string) (domain.FileStat, error) {
	if s == "" {
		return domain.FileStat{}, errors.New("fingerprint required")
	}
	sizeStr, mtimeStr, found := strings.Cut(s, "-")
	if !found {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint format: %q", s)
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint size: %w", err)
	}
	nanos, err := strconv.ParseInt(mtimeStr, 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint timestamp: %w", err)
	}
	var modTime time.Time
	if nanos != 0 {
		modTime = time.Unix(0, nanos)
	}
	return domain.FileStat{Path: "", Kind: "", Size: size, ModTime: modTime}, nil
}
