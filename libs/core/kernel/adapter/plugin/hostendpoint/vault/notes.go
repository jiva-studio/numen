package vault

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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

func (e *lookupsCalls) GetNoteRefs(
	ctx context.Context, req *connect.Request[v1.GetNoteRefsRequest],
) (*connect.Response[v1.GetNoteRefsResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNoteRefsResponse{Error: noVault}), nil
	}
	target, err := vault.Notes.GetNotes(ctx, req.Msg.GetPaths())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetNoteRefsResponse{Error: code}), err
	}
	notes := make([]*v1.HostNoteRef, 0, len(target))
	for _, path := range req.Msg.GetPaths() {
		if one, found := target[path]; found {
			notes = append(notes, &v1.HostNoteRef{Path: one.Path, Title: one.Title, Id: one.ID})
		}
	}
	return connect.NewResponse(&v1.GetNoteRefsResponse{Notes: notes}), nil
}

func (e *lookupsCalls) GetNoteTypes(
	ctx context.Context, req *connect.Request[v1.GetNoteTypesRequest],
) (*connect.Response[v1.GetNoteTypesResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNoteTypesResponse{Error: noVault}), nil
	}
	target, err := vault.Notes.GetTypes(ctx, req.Msg.GetPaths())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetNoteTypesResponse{Error: code}), err
	}
	types := make(map[string]string, len(target))
	for path, one := range target {
		types[path] = string(one)
	}
	return connect.NewResponse(&v1.GetNoteTypesResponse{Types: types}), nil
}

func (e *lookupsCalls) GetNamedPaths(
	ctx context.Context, req *connect.Request[v1.GetNamedPathsRequest],
) (*connect.Response[v1.GetNamedPathsResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNamedPathsResponse{Error: noVault}), nil
	}
	paths, err := vault.Notes.GetNamedPaths(ctx, req.Msg.GetName())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetNamedPathsResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetNamedPathsResponse{Paths: paths}), nil
}

func (e *lookupsCalls) GetPathsOfType(
	ctx context.Context, req *connect.Request[v1.GetPathsOfTypeRequest],
) (*connect.Response[v1.GetPathsOfTypeResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetPathsOfTypeResponse{Error: noVault}), nil
	}
	paths, err := vault.Notes.GetPathsOfType(ctx, domain.NoteType(req.Msg.GetType()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetPathsOfTypeResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetPathsOfTypeResponse{Paths: paths}), nil
}

func (e *lookupsCalls) GetLinks(
	ctx context.Context, req *connect.Request[v1.GetLinksRequest],
) (*connect.Response[v1.GetLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetLinksResponse{Error: noVault}), nil
	}
	links, err := vault.Links.GetLinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetLinksResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetLinksResponse{Links: writeLinks(links)}), nil
}

func (e *lookupsCalls) GetBacklinks(
	ctx context.Context, req *connect.Request[v1.GetBacklinksRequest],
) (*connect.Response[v1.GetBacklinksResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetBacklinksResponse{Error: noVault}), nil
	}
	links, err := vault.Links.GetBacklinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetBacklinksResponse{Error: code}), err
	}
	withRoles := writeLinks(links)
	for i, l := range links {
		inverse, _ := domain.GetInverseRole(l.Role)
		withRoles[i].SourceRole = wire.ConvertRoleToWire(inverse)
	}
	return connect.NewResponse(&v1.GetBacklinksResponse{Links: withRoles}), nil
}

func (e *lookupsCalls) ResolveLinks(
	ctx context.Context, req *connect.Request[v1.ResolveLinksRequest],
) (*connect.Response[v1.ResolveLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
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
