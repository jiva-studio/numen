package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

var errNoVault = errors.New("the call names no vault")

// Tools builds the registry of the vault tools.
type Tools struct {
	vaults  func(domain.VaultID) Vault
	library func(domain.VaultID) Library
	list    ports.VaultList
}

// Vault is the calls the tools make on one vault, each through the narrow interface it needs.
type Vault struct {
	Files    ports.FileReader
	Writer   ports.FileWriter
	Index    ports.Indexer
	Notes    ports.NoteLookup
	Links    ports.LinkLookup
	Arranger ports.NoteArranger
	Editor   ports.LinkEditor
	Selector ports.VaultSelector
}

// Library is the calls the library tools make on the text versions and the search index of one vault,
// each through the narrow interface it needs.
type Library struct {
	Versions ports.VersionLister
	Texts    ports.TextReader
	Search   ports.PassageSearch
}

// New creates a Tools instance. vaults gives the calls on the vault with an identity, and on the vault
// the host shows for the empty identity.
func New(vaults func(domain.VaultID) Vault) *Tools {
	return &Tools{vaults: vaults, library: nil, list: nil}
}

// WithVaultList sets the installation's list of vaults, which vault_list reads and the
// tools that add, rename, remove and open vaults change.
func (t *Tools) WithVaultList(list ports.VaultList) *Tools {
	t.list = list
	return t
}

// WithLibrary sets the calls on the text versions and the search index of the vault with an identity,
// which the library tools use.
func (t *Tools) WithLibrary(library func(domain.VaultID) Library) *Tools {
	t.library = library
	return t
}

// Registry registers and returns all vault tools.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	registrations := joinRegistrations(
		createNoteTools(), createLibraryTools(), createInspectTools(),
		createFileTools(), createArrangeTools(), createLinkEditTools(),
	)
	if t.list != nil {
		registrations = append(registrations, createVaultListTools()...)
	}
	for _, register := range registrations {
		if err := register(t, registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

type registration func(*Tools, *tools.Registry) error

func joinRegistrations(groups ...[]registration) []registration {
	var all []registration
	for _, group := range groups {
		all = append(all, group...)
	}
	return all
}

func bind[U, In, Out any](spec tools.Spec, use func(*call) U, run func(U, context.Context, In) (Out, error)) registration {
	return func(t *Tools, registry *tools.Registry) error {
		return tools.Add(registry, spec, func(ctx context.Context, in In) (Out, error) {
			c, err := t.open(ctx)
			if err != nil {
				var none Out
				return none, err
			}
			return run(use(c), ctx, in)
		})
	}
}
