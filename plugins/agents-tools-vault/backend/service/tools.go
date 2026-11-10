package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

var errNoVault = errors.New("the call names no vault")

// Tools provides MCP tools for AI agents interacting with the knowledge vault.
type Tools struct {
	vaults func(domain.VaultID) Vault
	vault  Vault
	texts  host.Texts
	search host.Search
	list   host.VaultList
}

// Vault is the calls the tools make on one vault, each through the narrow interface it needs.
type Vault struct {
	Files    host.FileReader
	Writer   host.FileWriter
	Index    host.VaultIndex
	Notes    host.NoteLookup
	Links    host.LinkLookup
	Arranger host.NoteArranger
	Editor   host.LinkEditor
	Selector host.VaultSelector
}

// New creates a Tools instance. vaults gives the calls on the vault with an identity, and on the vault
// the host shows for the empty identity.
func New(vaults func(domain.VaultID) Vault) *Tools {
	return &Tools{vaults: vaults, vault: Vault{}, texts: nil, search: nil, list: nil}
}

// WithVaultList sets the installation's list of vaults, which vault_list reads and the
// tools that add, rename, remove and open vaults change.
func (t *Tools) WithVaultList(list host.VaultList) *Tools {
	t.list = list
	return t
}

func inVault[In, Out any](t *Tools, run func(*Tools, context.Context, In) (Out, error)) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, in In) (Out, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			var none Out
			return none, errNoVault
		}
		call := &Tools{vaults: t.vaults, vault: t.vaults(domain.VaultID(id)), texts: nil, search: nil, list: t.list}
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
	adds := []func(*tools.Registry) error{
		t.addNoteRead,
		t.addNoteWrite,
		t.addNoteHeadingList,
		t.addNoteSearch,
		t.addNoteList,
		t.addNoteTitleList,
		t.addNoteResolve,
		t.addLibrarySearch,
		t.addLibraryRead,
		t.addLinkList,
		t.addVaultGet,
		t.addVaultList,
		t.addVaultProblemList,
		t.addFileRead,
		t.addFileCreate,
		t.addNoteRename,
		t.addNoteMove,
		t.addNoteRemove,
		t.addLinkAdd,
		t.addLinkUpdate,
		t.addLinkRemove,
	}
	if t.list != nil {
		adds = append(adds, t.addVaultAdd, t.addVaultRename, t.addVaultRemove, t.addVaultOpen)
	}
	for _, add := range adds {
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
		return domain.FileStat{}, errors.New("fingerprint is required")
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

func addTool[In, Out any](registry *tools.Registry, spec tools.Spec, run func(context.Context, In) (Out, error)) error {
	return tools.Add(registry, spec, run)
}
