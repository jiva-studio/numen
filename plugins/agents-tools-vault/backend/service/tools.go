package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

var errNoVault = errors.New("the call names no vault")

// Tools provides MCP tools for AI agents interacting with the knowledge vault.
type Tools struct {
	// host reaches every vault the host knows.
	host host.Vault
	// vault is the vault of the call being served.
	vault host.Vault
}

// New creates a Tools instance.
func New(vault host.Vault) *Tools {
	return &Tools{host: vault}
}

// inVault runs the tool on the vault the call names. A call that names none is refused.
func inVault[In, Out any](t *Tools, run func(*Tools, context.Context, In) (Out, error)) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, in In) (Out, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			var none Out
			return none, errNoVault
		}
		return run(&Tools{host: t.host, vault: t.host.With(domain.VaultID(id))}, ctx, in)
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
		t.addSearchQuery,
		t.addNoteNeighbourhood,
		t.addLinkList,
		t.addVaultInfo,
		t.addVaultGet,
		t.addFileRead,
		t.addFileCreate,
	} {
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func formatFingerprint(ref domain.Fingerprint) string {
	var nanos int64
	if !ref.ModTime.IsZero() {
		nanos = ref.ModTime.UnixNano()
	}
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(nanos, 10)
}

func parseFingerprint(s string) (domain.Fingerprint, error) {
	if s == "" {
		return domain.Fingerprint{}, errors.New("fingerprint required")
	}
	sizeStr, mtimeStr, found := strings.Cut(s, "-")
	if !found {
		return domain.Fingerprint{}, fmt.Errorf("invalid fingerprint format: %q", s)
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("invalid fingerprint size: %w", err)
	}
	nanos, err := strconv.ParseInt(mtimeStr, 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("invalid fingerprint timestamp: %w", err)
	}
	var modTime time.Time
	if nanos != 0 {
		modTime = time.Unix(0, nanos)
	}
	return domain.Fingerprint{Size: size, ModTime: modTime}, nil
}
