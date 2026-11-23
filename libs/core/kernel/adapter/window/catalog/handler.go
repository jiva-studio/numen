// Package catalog answers a window's CatalogService: the plugins the application has, the ones
// it could have, and the plugin that provides a kind of tab.
package catalog

import (
	"context"
	"errors"
	"os"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Catalog is the plugins the application lists and finds.
type Catalog interface {
	List(ctx context.Context) (catalog.Listing, error)
	Get(ctx context.Context, id string) (catalog.Entry, error)
	FindByTabKind(ctx context.Context, kind string) (catalog.Entry, error)
}

// InstalledPlugins lists the installed plugins, enabled or not.
type InstalledPlugins func() []manifest.Discovered

// Handler answers CatalogService over a catalog and the installed plugins.
type Handler struct {
	plugins   Catalog
	installed InstalledPlugins
}

// NewHandler returns the handler over the catalog and the installed plugins. It panics on a nil installed: an
// installation with no plugin folders lists none.
func NewHandler(plugins Catalog, installed InstalledPlugins) *Handler {
	if installed == nil {
		panic("catalog: installed plugins are required") //nolint:forbidigo // a nil here is a wiring bug, and a constructor of an adapter returns no error
	}
	return &Handler{plugins: plugins, installed: installed}
}

// readmeName is the file of a plugin's folder that describes the plugin.
const readmeName = "README.md"

// GetCatalogReadme returns the README.md of an installed plugin. A plugin with no folder, or whose folder holds
// no README.md, has an empty text; no plugin is started.
func (h *Handler) GetCatalogReadme(
	ctx context.Context, r *connect.Request[v1.GetCatalogReadmeRequest],
) (*connect.Response[v1.GetCatalogReadmeResponse], error) {
	dir, isInstalled := h.getPluginDir(r.Msg.GetId())
	if !isInstalled {
		return connect.NewResponse(&v1.GetCatalogReadmeResponse{}), nil
	}
	text, err := wire.ReadPluginFile(ctx, dir, readmeName)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, os.ErrNotExist) {
		return connect.NewResponse(&v1.GetCatalogReadmeResponse{}), nil
	}
	if err != nil {
		return nil, wire.NewInternalError(err)
	}
	return connect.NewResponse(&v1.GetCatalogReadmeResponse{Text: string(text)}), nil
}

// ListCatalog lists the entries after the merge. A source that failed is listed in source_errors and
// does not hide the entries of the others.
func (h *Handler) ListCatalog(
	ctx context.Context, _ *connect.Request[v1.ListCatalogRequest],
) (*connect.Response[v1.ListCatalogResponse], error) {
	listing, err := h.plugins.List(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if err != nil && len(listing.Errors) == 0 {
		return nil, wire.NewInternalError(err)
	}
	entries := make([]*v1.CatalogEntry, 0, len(listing.Entries))
	for _, one := range listing.Entries {
		entries = append(entries, newWireEntry(one))
	}
	return connect.NewResponse(&v1.ListCatalogResponse{Entries: entries, SourceErrors: newWireSourceErrors(listing.Errors)}), nil
}

// GetCatalogEntry returns the entry of one plugin, or NOT_FOUND when no source lists it.
func (h *Handler) GetCatalogEntry(
	ctx context.Context, r *connect.Request[v1.GetCatalogEntryRequest],
) (*connect.Response[v1.GetCatalogEntryResponse], error) {
	entry, err := h.plugins.Get(ctx, r.Msg.GetId())
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	sourceErrors := newWireSourceErrors(catalog.FindSourceErrors(err))
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewResponse(&v1.GetCatalogEntryResponse{Error: newWireErrorCode(err), SourceErrors: sourceErrors}), nil
	}
	if err != nil && len(sourceErrors) == 0 {
		return nil, wire.NewInternalError(err)
	}
	return connect.NewResponse(&v1.GetCatalogEntryResponse{Entry: newWireEntry(entry), SourceErrors: sourceErrors}), nil
}

// FindCatalogEntryByTabKind returns the entry of the plugin that provides the kind of tab, or
// NOT_FOUND when none does.
func (h *Handler) FindCatalogEntryByTabKind(
	ctx context.Context, r *connect.Request[v1.FindCatalogEntryByTabKindRequest],
) (*connect.Response[v1.FindCatalogEntryByTabKindResponse], error) {
	entry, err := h.plugins.FindByTabKind(ctx, r.Msg.GetKind())
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	sourceErrors := newWireSourceErrors(catalog.FindSourceErrors(err))
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewResponse(&v1.FindCatalogEntryByTabKindResponse{Error: newWireErrorCode(err), SourceErrors: sourceErrors}), nil
	}
	if err != nil && len(sourceErrors) == 0 {
		return nil, wire.NewInternalError(err)
	}
	return connect.NewResponse(&v1.FindCatalogEntryByTabKindResponse{Entry: newWireEntry(entry), SourceErrors: sourceErrors}), nil
}

func newWireSourceErrors(failures []*catalog.SourceError) []*v1.CatalogSourceError {
	if len(failures) == 0 {
		return nil
	}
	out := make([]*v1.CatalogSourceError, 0, len(failures))
	for _, failure := range failures {
		out = append(out, &v1.CatalogSourceError{Source: failure.Source, Message: failure.Err.Error()})
	}
	return out
}

func newWireEntry(entry catalog.Entry) *v1.CatalogEntry {
	wire := &v1.CatalogEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Version:     entry.Version,
		Description: entry.Description,
		Category:    entry.Category,
		Contributes: newWireContributes(entry.Contributes),
		Source:      entry.Source,
		State:       newWireState(entry.State),
	}
	if entry.Incompatibility != nil {
		wire.Incompatibility = newWireIncompatibility(entry.Incompatibility.Reason)
		wire.IncompatibilityDetail = entry.Incompatibility.Detail
	}
	return wire
}

func newWireContributes(contributes manifest.Contributes) *v1.ManifestContributes {
	kinds := make([]*v1.ManifestTabKind, 0, len(contributes.TabKinds))
	for _, kind := range contributes.TabKinds {
		kinds = append(kinds, &v1.ManifestTabKind{Kind: kind.Kind, Title: kind.Title, Icon: kind.Icon})
	}
	return &v1.ManifestContributes{TabKinds: kinds, Feature: contributes.Feature, Category: contributes.Category}
}

func newWireState(state catalog.State) v1.CatalogState {
	switch state {
	case catalog.StateInstalled:
		return v1.CatalogState_CATALOG_STATE_INSTALLED
	case catalog.StateDisabled:
		return v1.CatalogState_CATALOG_STATE_DISABLED
	case catalog.StateAvailable:
		return v1.CatalogState_CATALOG_STATE_AVAILABLE
	}
	return v1.CatalogState_CATALOG_STATE_UNSPECIFIED
}

func newWireIncompatibility(reason manifest.IncompatibleReason) v1.CatalogIncompatibility {
	switch reason {
	case manifest.ReasonTarget:
		return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_TARGET
	case manifest.ReasonProtocol:
		return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_PROTOCOL
	}
	return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_UNSPECIFIED
}

func newWireErrorCode(err error) v1.CatalogErrorCode {
	if errors.Is(err, catalog.ErrNotFound) {
		return v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND
	}
	return v1.CatalogErrorCode_CATALOG_ERROR_CODE_UNSPECIFIED
}

func (h *Handler) getPluginDir(id string) (string, bool) {
	for _, one := range h.installed() {
		if one.Manifest.ID == id {
			return one.Dir, true
		}
	}
	return "", false
}
