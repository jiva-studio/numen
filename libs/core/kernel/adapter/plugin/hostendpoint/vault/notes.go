package vault

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func writeLink(l domain.ResolvedLink) *v1.HostLink {
	return &v1.HostLink{
		From:         l.From,
		To:           l.To,
		ToVault:      string(l.ToVault),
		LinkType:     l.Type,
		TargetScheme: l.Target.Scheme,
		TargetValue:  l.Target.Value,
		Label:        l.Label,
		IsAmbiguous:  l.IsAmbiguous,
		Role:         wire.ConvertRoleToWire(l.Role),
		Note:         l.Why,
		Marker:       wire.ConvertMarkerToWire(l.Marker),
	}
}

func writeLinks(links []domain.ResolvedLink) []*v1.HostLink {
	out := make([]*v1.HostLink, 0, len(links))
	for _, one := range links {
		out = append(out, writeLink(one))
	}
	return out
}

func (e *lookupsCalls) ListNoteRefs(
	ctx context.Context, req *connect.Request[v1.ListNoteRefsRequest],
) (*connect.Response[v1.ListNoteRefsResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListNoteRefsResponse{Error: noVault}), nil
	}
	target, err := vault.Notes.GetNotes(ctx, req.Msg.GetPaths())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListNoteRefsResponse{Error: code}), err
	}
	notes := make([]*v1.Note, 0, len(target))
	for _, path := range req.Msg.GetPaths() {
		if one, found := target[path]; found {
			notes = append(notes, &v1.Note{Path: one.Path, Title: one.Title, Identifier: one.ID})
		}
	}
	return connect.NewResponse(&v1.ListNoteRefsResponse{Notes: notes}), nil
}

func (e *lookupsCalls) ListNoteTypes(
	ctx context.Context, req *connect.Request[v1.ListNoteTypesRequest],
) (*connect.Response[v1.ListNoteTypesResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListNoteTypesResponse{Error: noVault}), nil
	}
	target, err := vault.Notes.GetTypes(ctx, req.Msg.GetPaths())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListNoteTypesResponse{Error: code}), err
	}
	types := make(map[string]string, len(target))
	for path, one := range target {
		types[path] = string(one)
	}
	return connect.NewResponse(&v1.ListNoteTypesResponse{Types: types}), nil
}

func (e *lookupsCalls) ListNamedPaths(
	ctx context.Context, req *connect.Request[v1.ListNamedPathsRequest],
) (*connect.Response[v1.ListNamedPathsResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListNamedPathsResponse{Error: noVault}), nil
	}
	paths, err := vault.Notes.ListNamedPaths(ctx, req.Msg.GetName())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListNamedPathsResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ListNamedPathsResponse{Paths: paths}), nil
}

func (e *lookupsCalls) ListPathsOfType(
	ctx context.Context, req *connect.Request[v1.ListPathsOfTypeRequest],
) (*connect.Response[v1.ListPathsOfTypeResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListPathsOfTypeResponse{Error: noVault}), nil
	}
	paths, err := vault.Notes.ListPathsOfType(ctx, domain.NoteType(req.Msg.GetType()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListPathsOfTypeResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ListPathsOfTypeResponse{Paths: paths}), nil
}

func (e *lookupsCalls) ListLinks(
	ctx context.Context, req *connect.Request[v1.ListLinksRequest],
) (*connect.Response[v1.ListLinksResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListLinksResponse{Error: noVault}), nil
	}
	links, err := vault.Links.ListLinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListLinksResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ListLinksResponse{Links: writeLinks(links)}), nil
}

func (e *lookupsCalls) ListBacklinks(
	ctx context.Context, req *connect.Request[v1.HostLinkServiceListBacklinksRequest],
) (*connect.Response[v1.HostLinkServiceListBacklinksResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostLinkServiceListBacklinksResponse{Error: noVault}), nil
	}
	links, err := vault.Links.ListBacklinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostLinkServiceListBacklinksResponse{Error: code}), err
	}
	withRoles := writeLinks(links)
	for i, l := range links {
		inverse, _ := domain.GetInverseRole(l.Role)
		withRoles[i].SourceRole = wire.ConvertRoleToWire(inverse)
	}
	return connect.NewResponse(&v1.HostLinkServiceListBacklinksResponse{Links: withRoles}), nil
}

func (e *lookupsCalls) ResolveLinks(
	ctx context.Context, req *connect.Request[v1.ResolveLinksRequest],
) (*connect.Response[v1.ResolveLinksResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ResolveLinksResponse{Error: noVault}), nil
	}
	resolvedLinks, err := vault.Links.ResolveLinks(ctx, req.Msg.GetPath(), req.Msg.GetAddresses())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ResolveLinksResponse{Error: code}), err
	}
	links := make(map[string]*v1.HostLink, len(resolvedLinks))
	for content, one := range resolvedLinks {
		links[content] = writeLink(one)
	}
	return connect.NewResponse(&v1.ResolveLinksResponse{Links: links}), nil
}
