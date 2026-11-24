package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

var errNoVault = errors.New("the call names no vault")

// Tools builds the registry of the vault tools.
type Tools struct {
	getVault           func(domain.VaultID) Vault
	getLibrary         func(domain.VaultID) Library
	functions          host.SearchFunctions
	getSearchFunctions func(ctx context.Context) (host.SearchFunctions, error)
	list               ports.VaultList
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
	Search   ports.TextSearch
	Queries  ports.QueryRunner
}

// New creates a Tools instance. vaults gives the calls on the vault with an identity, and on the vault
// the host shows for the empty identity.
func New(vaults func(domain.VaultID) Vault) *Tools {
	return &Tools{getVault: vaults, getLibrary: nil, functions: host.SearchFunctions{Functions: nil, Revision: ""}, getSearchFunctions: nil, list: nil}
}

// WithVaultList sets the installation's list of vaults, which vault_list reads and the
// tools that add, rename, remove and open vaults change.
func (t *Tools) WithVaultList(list ports.VaultList) *Tools {
	t.list = list
	return t
}

// WithLibrary sets the calls on the text versions and the search index of the vault with an identity,
// which the library tools use, and the functions of the search language library_query describes.
func (t *Tools) WithLibrary(library func(domain.VaultID) Library, functions host.SearchFunctions) *Tools {
	t.getLibrary, t.functions = library, functions
	return t
}

// WithFunctionLister sets how library_query reads the functions of the search language each time the host
// lists the tools, so that its description follows the functions of the plugins enabled since the start.
func (t *Tools) WithFunctionLister(lister func(ctx context.Context) (host.SearchFunctions, error)) *Tools {
	t.getSearchFunctions = lister
	return t
}

// BuildRegistry registers and returns all vault tools.
func (t *Tools) BuildRegistry() (*tools.Registry, error) {
	registry := tools.New()
	registrations := joinRegistrations(
		createNoteTools(), createLibraryTools(), createQueryTools(t.functions, t.getSearchFunctions), createInspectTools(),
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

func bind[U, In, Out any](spec tools.Spec, use func(*call) U, run func(context.Context, U, In) (Out, error)) registration {
	return func(t *Tools, registry *tools.Registry) error {
		return tools.Add(registry, spec, func(ctx context.Context, in In) (Out, error) {
			c, err := t.open(ctx)
			if err != nil {
				var none Out
				return none, err
			}
			return run(ctx, use(c), in)
		})
	}
}
