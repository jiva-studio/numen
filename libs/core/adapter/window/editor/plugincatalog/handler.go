// Package plugincatalog answers a window's CatalogService: the plugins the application has, the ones
// it could have, and the plugin that provides a kind of tab.
package plugincatalog

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// Catalog is the plugins the application lists and finds.
type Catalog = interface {
	List(ctx context.Context) (catalog.Listing, error)
	Get(ctx context.Context, id string) (catalog.Entry, error)
	FindByTabKind(ctx context.Context, kind string) (catalog.Entry, error)
}

// Handler answers CatalogService over a catalog.
type Handler struct {
	plugins Catalog
}

// NewHandler returns the handler over the catalog.
func NewHandler(plugins Catalog) *Handler {
	return &Handler{plugins: plugins}
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
		return nil, connect.NewError(connect.CodeInternal, err)
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
	sourceErrors := newWireSourceErrors(catalog.SourceErrors(err))
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewResponse(&v1.GetCatalogEntryResponse{Error: newWireErrorCode(err), SourceErrors: sourceErrors}), nil
	}
	if err != nil && len(sourceErrors) == 0 {
		return nil, connect.NewError(connect.CodeInternal, err)
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
	sourceErrors := newWireSourceErrors(catalog.SourceErrors(err))
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewResponse(&v1.FindCatalogEntryByTabKindResponse{Error: newWireErrorCode(err), SourceErrors: sourceErrors}), nil
	}
	if err != nil && len(sourceErrors) == 0 {
		return nil, connect.NewError(connect.CodeInternal, err)
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
	case manifest.ReasonAppTooOld:
		return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_APP_TOO_OLD
	case manifest.ReasonAppTooNew:
		return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_APP_TOO_NEW
	}
	return v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_UNSPECIFIED
}

// newWireErrorCode is the code of a call served without an entry. An error that is no refusal to
// find the entry has no code.
func newWireErrorCode(err error) v1.CatalogErrorCode {
	if errors.Is(err, catalog.ErrNotFound) {
		return v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND
	}
	return v1.CatalogErrorCode_CATALOG_ERROR_CODE_UNSPECIFIED
}
