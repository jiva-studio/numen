package hostendpoint

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var errTooManyLinks = fmt.Errorf("too many links: at most %d per call", batchLimit)

// readRole returns the role of a link on the wire, empty for ROLE_UNSPECIFIED. A value the schema does
// not name is kept as its number, which no link has, so the core rejects it.
func readRole(role v1.Role) domain.LinkRole {
	if converted, isKnown := wire.ConvertRoleFromWire(role); isKnown || role == v1.Role_ROLE_UNSPECIFIED {
		return converted
	}
	return domain.LinkRole(role.String())
}

func (e *vaultEndpoint) AddLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceAddLinksRequest],
) (*connect.Response[v1.HostVaultServiceAddLinksResponse], error) {
	if len(req.Msg.GetLinks()) > batchLimit {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyLinks)
	}
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceAddLinksResponse{Error: noVault}), nil
	}
	additions := make([]pluginport.PluginLinkAddition, len(req.Msg.GetLinks()))
	for i, one := range req.Msg.GetLinks() {
		additions[i] = pluginport.PluginLinkAddition{From: one.GetFrom(), Baseline: readFileStat(one.GetBaseline()), Link: domain.Link{
			Target: domain.ParseAddress(one.GetTo()), Role: readRole(one.GetRole()),
			Type: one.GetType(), Label: one.GetLabel(), Why: one.GetNote(),
		}}
	}
	added, err := vault.AddLinks(ctx, additions)
	if err != nil {
		_, code, callErr := splitNoteError(ctx, err)
		if callErr == nil && code == 0 {
			callErr = connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.HostVaultServiceAddLinksResponse{Error: code}), callErr
	}
	out := &v1.HostVaultServiceAddLinksResponse{Results: make([]*v1.HostLinkAdded, len(added))}
	for i, one := range added {
		entry, code, callErr := writeLinkEdit(ctx, one.Stat, one.Err)
		if callErr != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceAddLinksResponse{Error: code}), callErr
		}
		out.Results[i] = &v1.HostLinkAdded{Stat: entry.stat, Error: entry.code, Detail: entry.detail}
	}
	return connect.NewResponse(out), nil
}

func (e *vaultEndpoint) UpdateLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceUpdateLinksRequest],
) (*connect.Response[v1.HostVaultServiceUpdateLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceUpdateLinksResponse{Error: noVault}), nil
	}
	m := req.Msg
	change := domain.Link{Role: readRole(m.GetRole()), Type: m.GetType(), Label: m.GetLabel(), Why: m.GetNote()}
	stat, err := vault.UpdateLinks(ctx, m.GetFrom(), domain.ParseAddress(m.GetTo()), change, readFileStat(m.GetBaseline()))
	entry, code, callErr := writeLinkEdit(ctx, stat, err)
	if callErr != nil || code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceUpdateLinksResponse{Error: code}), callErr
	}
	return connect.NewResponse(&v1.HostVaultServiceUpdateLinksResponse{Stat: entry.stat, Error: entry.code, Detail: entry.detail}), nil
}

func (e *vaultEndpoint) RemoveLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveLinksRequest],
) (*connect.Response[v1.HostVaultServiceRemoveLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveLinksResponse{Error: noVault}), nil
	}
	m := req.Msg
	stat, err := vault.RemoveLinks(ctx, m.GetFrom(), domain.ParseAddress(m.GetTo()), readRole(m.GetRole()), readFileStat(m.GetBaseline()))
	entry, code, callErr := writeLinkEdit(ctx, stat, err)
	if callErr != nil || code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRemoveLinksResponse{Error: code}), callErr
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveLinksResponse{Stat: entry.stat, Error: entry.code, Detail: entry.detail}), nil
}

// linkEdit is the outcome of an edit of the links of one note on the wire.
type linkEdit struct {
	stat   *v1.HostFileStat
	code   v1.HostVaultErrorCode
	detail string
}

// writeLinkEdit returns the outcome of an edit of the links of one note: the file stat after the write,
// or the code and the core's text of the failure. A failure of the call is returned as the call's
// code or error.
func writeLinkEdit(ctx context.Context, stat domain.FileStat, err error) (linkEdit, v1.HostVaultErrorCode, error) {
	if err == nil {
		return linkEdit{stat: writeFileStat(stat)}, 0, nil
	}
	noteCode, code, callErr := splitNoteError(ctx, err)
	if callErr != nil || code != 0 {
		return linkEdit{}, code, callErr
	}
	return linkEdit{code: noteCode, detail: err.Error()}, 0, nil
}
