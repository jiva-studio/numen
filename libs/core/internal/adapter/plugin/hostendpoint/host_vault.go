package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

// vaultLockHold is the longest a plugin holds the writers of the vault. A plugin that goes without
// letting them go must not stop the vault being written.
const vaultLockHold = 2 * time.Minute

const vaultWatchPoll = time.Second

// vaultHeader is the header of a call that names the vault it works, by identity.
const vaultHeader = "Numen-Vault"

// vaultEndpoint answers HostVaultService for the plugins that call the host.
type vaultEndpoint struct {
	endpoint *Endpoint

	mu    sync.Mutex
	locks map[string]func()
	next  int
}

func newVaultEndpoint(endpoint *Endpoint) *vaultEndpoint {
	return &vaultEndpoint{endpoint: endpoint, locks: map[string]func(){}}
}

var _ numenv1connect.HostVaultServiceHandler = (*vaultEndpoint)(nil)

// begin authorizes a call and returns the plugin that made it and the vault it works. A plugin that
func (e *vaultEndpoint) begin(header http.Header, permission string) (string, pluginport.PluginVault, error) {
	pluginID, err := e.endpoint.authorize(header.Get("Authorization"), permission)
	if err != nil {
		return "", nil, err
	}
	vault, err := e.endpoint.getVault(header)
	if err != nil {
		return "", nil, err
	}
	return pluginID, vault, nil
}

// getVault is the vault a call works: the one its Numen-Vault header names, else the one the host
// shows. It is nil where the host has no vault to offer.
func (e *Endpoint) getVault(header http.Header) (pluginport.PluginVault, error) {
	vault := e.callers.GetPluginVault()
	named := header.Get(vaultHeader)
	if vault == nil || named == "" {
		return vault, nil
	}
	linked, err := vault.With(domain.VaultID(named))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return linked, nil
}

// answer, the end of the caller's context is cancelled, and anything else is internal.
func answer(ctx context.Context, err error) (v1.HostVaultErrorCode, error) {
	if err == nil {
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, hostvault.ErrNoVault):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT, nil
	case errors.Is(err, vaultport.ErrStale):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_STALE, nil
	case errors.Is(err, vaultport.ErrAlreadyExists):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS, nil
	case errors.Is(err, vaultport.ErrLockedByAnother):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_LOCKED_BY_ANOTHER, nil
	case errors.Is(err, vaultport.ErrNotANote):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE, nil
	case errors.Is(err, vaultport.ErrOutside):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE, nil
	case errors.Is(err, domain.ErrHierarchyCycle):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_HIERARCHY_CYCLE, nil
	case errors.Is(err, domain.ErrInvalidConnection), errors.Is(err, domain.ErrNotCreatable):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION, nil
	case errors.Is(err, domain.ErrNoFreeName):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NAME_EXHAUSTED, nil
	case errors.Is(err, fs.ErrNotExist):
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_MISSING, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

const noVault = v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT

func writeFileStat(f domain.FileStat) *v1.HostFileStat {
	return &v1.HostFileStat{Path: f.Path, Size: f.Size, ModTime: f.ModTime.UnixNano(), IsNote: f.Kind == domain.KindNote}
}

func readFileStat(f *v1.HostFileStat) domain.FileStat {
	if f == nil {
		return domain.FileStat{}
	}
	return domain.FileStat{
		Path: f.GetPath(), Kind: readKind(f), Size: f.GetSize(), ModTime: time.Unix(0, f.GetModTime()),
	}
}

// readKind returns the source kind for the file that a file stat describes, or
// none for a file stat with no file.
func readKind(f *v1.HostFileStat) domain.SourceKind {
	switch {
	case f.GetIsNote():
		return domain.KindNote
	case f.GetPath() == "":
		return ""
	default:
		return domain.KindFile
	}
}

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

func (e *vaultEndpoint) GetHostVault(
	ctx context.Context, req *connect.Request[v1.GetHostVaultRequest],
) (*connect.Response[v1.GetHostVaultResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetHostVaultResponse{Error: noVault}), nil
	}
	v, showing := vault.GetVault()
	if !showing {
		return connect.NewResponse(&v1.GetHostVaultResponse{Error: noVault}), nil
	}
	return connect.NewResponse(&v1.GetHostVaultResponse{Path: v.Path, Id: string(v.ID), Name: v.Name}), nil
}

func (e *vaultEndpoint) ListHostVaults(
	_ context.Context, req *connect.Request[v1.ListHostVaultsRequest],
) (*connect.Response[v1.ListHostVaultsResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListHostVaultsResponse{}), nil
	}
	recognized, err := vault.GetVaults()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.HostVaultInfo, 0, len(recognized))
	for _, one := range recognized {
		out = append(out, &v1.HostVaultInfo{Id: string(one.ID), Name: one.Name, Path: one.Path})
	}
	return connect.NewResponse(&v1.ListHostVaultsResponse{Vaults: out}), nil
}

func (e *vaultEndpoint) EnsureVaultIndexed(
	ctx context.Context, req *connect.Request[v1.EnsureVaultIndexedRequest],
) (*connect.Response[v1.EnsureVaultIndexedResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.EnsureVaultIndexedResponse{Error: noVault}), nil
	}
	state, err := vault.EnsureIndexed(ctx)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.EnsureVaultIndexedResponse{Error: code}), err
	}
	out := &v1.EnsureVaultIndexedResponse{Reason: state.Reason}
	switch state.Status {
	case pluginport.PluginIndexed:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_INDEXED
	case pluginport.PluginIndexing:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_INDEXING
	case pluginport.PluginIndexFailed:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_FAILED
	}
	return connect.NewResponse(out), nil
}

func (e *vaultEndpoint) WatchMoves(
	ctx context.Context, req *connect.Request[v1.WatchMovesRequest], stream *connect.ServerStream[v1.WatchMovesResponse],
) error {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return err
	}
	if vault == nil {
		return connect.NewError(connect.CodeFailedPrecondition, hostvault.ErrNoVault)
	}
	moves, err := vault.WatchMoves(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if err := stream.Send(&v1.WatchMovesResponse{}); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, open := <-moves:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.WatchMovesResponse{}); err != nil {
				return err
			}
		}
	}
}

func (e *vaultEndpoint) StatFile(
	ctx context.Context, req *connect.Request[v1.StatFileRequest],
) (*connect.Response[v1.StatFileResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.StatFileResponse{Error: noVault}), nil
	}
	pos, err := vault.StatFile(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.StatFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.StatFileResponse{Stat: writeFileStat(pos)}), nil
}

func (e *vaultEndpoint) ReadFile(
	ctx context.Context, req *connect.Request[v1.ReadFileRequest],
) (*connect.Response[v1.ReadFileResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ReadFileResponse{Error: noVault}), nil
	}
	content, pos, err := vault.Read(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadFileResponse{Content: content, Stat: writeFileStat(pos)}), nil
}

func (e *vaultEndpoint) WriteFile(
	ctx context.Context, req *connect.Request[v1.WriteFileRequest],
) (*connect.Response[v1.WriteFileResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteFileResponse{Error: noVault}), nil
	}
	pos, err := vault.Write(ctx, req.Msg.GetPath(), req.Msg.GetContent(), readFileStat(req.Msg.GetBaseline()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.WriteFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.WriteFileResponse{Stat: writeFileStat(pos)}), nil
}

func (e *vaultEndpoint) CreateFile(
	ctx context.Context, req *connect.Request[v1.CreateFileRequest],
) (*connect.Response[v1.CreateFileResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.CreateFileResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Create(ctx, req.Msg.GetPath(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.CreateFileResponse{Error: code}), err
}

func (e *vaultEndpoint) LockVault(
	ctx context.Context, req *connect.Request[v1.LockVaultRequest],
) (*connect.Response[v1.LockVaultResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.LockVaultResponse{Error: noVault}), nil
	}
	release, err := vault.Lock(ctx)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.LockVaultResponse{Error: code}), err
	}
	e.mu.Lock()
	e.next++
	id := fmt.Sprintf("lock-%d", e.next)
	e.locks[id] = release
	e.mu.Unlock()
	time.AfterFunc(vaultLockHold, func() { e.unlock(id) })
	return connect.NewResponse(&v1.LockVaultResponse{Lock: id}), nil
}

// unlock releases the writers once, on the first call.
func (e *vaultEndpoint) unlock(id string) {
	e.mu.Lock()
	release := e.locks[id]
	delete(e.locks, id)
	e.mu.Unlock()
	if release != nil {
		release()
	}
}

func (e *vaultEndpoint) UnlockVault(
	_ context.Context, req *connect.Request[v1.UnlockVaultRequest],
) (*connect.Response[v1.UnlockVaultResponse], error) {
	if _, _, err := e.begin(req.Header(), pluginport.PermissionVaultWrite); err != nil {
		return nil, err
	}
	e.unlock(req.Msg.GetLock())
	return connect.NewResponse(&v1.UnlockVaultResponse{}), nil
}

func (e *vaultEndpoint) ScanFiles(
	ctx context.Context, req *connect.Request[v1.ScanFilesRequest],
) (*connect.Response[v1.ScanFilesResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ScanFilesResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Scan(ctx, req.Msg.GetPaths()))
	return connect.NewResponse(&v1.ScanFilesResponse{Error: code}), err
}

func (e *vaultEndpoint) GetNoteRefs(
	ctx context.Context, req *connect.Request[v1.GetNoteRefsRequest],
) (*connect.Response[v1.GetNoteRefsResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNoteRefsResponse{Error: noVault}), nil
	}
	target, err := vault.GetNotes(ctx, req.Msg.GetPaths())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetNoteRefsResponse{Error: code}), err
	}
	notes := make([]*v1.HostNoteRef, 0, len(target))
	for _, path := range req.Msg.GetPaths() {
		if one, found := target[path]; found {
			notes = append(notes, &v1.HostNoteRef{Path: one.Path, Title: one.Title})
		}
	}
	return connect.NewResponse(&v1.GetNoteRefsResponse{Notes: notes}), nil
}

func (e *vaultEndpoint) GetNoteTypes(
	ctx context.Context, req *connect.Request[v1.GetNoteTypesRequest],
) (*connect.Response[v1.GetNoteTypesResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNoteTypesResponse{Error: noVault}), nil
	}
	target, err := vault.GetTypes(ctx, req.Msg.GetPaths())
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

func (e *vaultEndpoint) GetNamedPaths(
	ctx context.Context, req *connect.Request[v1.GetNamedPathsRequest],
) (*connect.Response[v1.GetNamedPathsResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetNamedPathsResponse{Error: noVault}), nil
	}
	paths, err := vault.GetNamedPaths(ctx, req.Msg.GetName())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetNamedPathsResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetNamedPathsResponse{Paths: paths}), nil
}

func (e *vaultEndpoint) GetPathsOfType(
	ctx context.Context, req *connect.Request[v1.GetPathsOfTypeRequest],
) (*connect.Response[v1.GetPathsOfTypeResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetPathsOfTypeResponse{Error: noVault}), nil
	}
	paths, err := vault.GetPathsOfType(ctx, domain.NoteType(req.Msg.GetType()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetPathsOfTypeResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetPathsOfTypeResponse{Paths: paths}), nil
}

func (e *vaultEndpoint) GetLinks(
	ctx context.Context, req *connect.Request[v1.GetLinksRequest],
) (*connect.Response[v1.GetLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetLinksResponse{Error: noVault}), nil
	}
	links, err := vault.GetLinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetLinksResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetLinksResponse{Links: writeLinks(links)}), nil
}

func (e *vaultEndpoint) GetBacklinks(
	ctx context.Context, req *connect.Request[v1.GetBacklinksRequest],
) (*connect.Response[v1.GetBacklinksResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetBacklinksResponse{Error: noVault}), nil
	}
	links, err := vault.GetBacklinks(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetBacklinksResponse{Error: code}), err
	}
	withRoles := writeLinks(links)
	for i, l := range links {
		inverse, _ := domain.GetInverseRole(l.Role)
		withRoles[i].RoleFromHere = wire.ConvertRoleToWire(inverse)
	}
	return connect.NewResponse(&v1.GetBacklinksResponse{Links: withRoles}), nil
}

func (e *vaultEndpoint) ResolveLinks(
	ctx context.Context, req *connect.Request[v1.ResolveLinksRequest],
) (*connect.Response[v1.ResolveLinksResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ResolveLinksResponse{Error: noVault}), nil
	}
	reached, err := vault.ResolveLinks(ctx, req.Msg.GetPath(), req.Msg.GetWritten())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ResolveLinksResponse{Error: code}), err
	}
	links := make(map[string]*v1.HostLink, len(reached))
	for content, one := range reached {
		links[content] = writeLink(one)
	}
	return connect.NewResponse(&v1.ResolveLinksResponse{Links: links}), nil
}

func (e *vaultEndpoint) GetParseProblems(
	ctx context.Context, req *connect.Request[v1.GetParseProblemsRequest],
) (*connect.Response[v1.GetParseProblemsResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetParseProblemsResponse{Error: noVault}), nil
	}
	target, err := vault.GetParseProblems(ctx)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetParseProblemsResponse{Error: code}), err
	}
	problems := make([]*v1.HostParseProblem, 0, len(target))
	for _, one := range target {
		problems = append(problems, &v1.HostParseProblem{Path: one.Path, Detail: one.Detail})
	}
	return connect.NewResponse(&v1.GetParseProblemsResponse{Problems: problems}), nil
}

func (e *vaultEndpoint) ReadCache(
	ctx context.Context, req *connect.Request[v1.ReadCacheRequest],
) (*connect.Response[v1.ReadCacheResponse], error) {
	pluginID, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ReadCacheResponse{Error: noVault}), nil
	}
	content, found, err := vault.ReadCache(ctx, pluginID, req.Msg.GetName())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadCacheResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadCacheResponse{Found: found, Content: content}), nil
}

func (e *vaultEndpoint) WriteCache(
	ctx context.Context, req *connect.Request[v1.WriteCacheRequest],
) (*connect.Response[v1.WriteCacheResponse], error) {
	pluginID, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteCacheResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.WriteCache(ctx, pluginID, req.Msg.GetName(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.WriteCacheResponse{Error: code}), err
}

func (e *vaultEndpoint) WatchVault(
	ctx context.Context, req *connect.Request[v1.WatchVaultRequest], stream *connect.ServerStream[v1.WatchVaultResponse],
) error {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return err
	}
	if vault == nil {
		return connect.NewError(connect.CodeFailedPrecondition, hostvault.ErrNoVault)
	}
	watched, showing := vault.GetVault()
	if !showing {
		return connect.NewError(connect.CodeFailedPrecondition, hostvault.ErrNoVault)
	}
	changes, missing, err := vault.Watch(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if err := stream.Send(&v1.WatchVaultResponse{}); err != nil {
		return err
	}
	return followVault(ctx, vault, watched, changes, missing, stream)
}

// followVault sends the changes of the vault until the watch ends, and reports
// when the window switches to another vault.
func followVault(
	ctx context.Context, vault pluginport.PluginVault, watched domain.Vault,
	changes <-chan []string, lostChan <-chan struct{}, stream *connect.ServerStream[v1.WatchVaultResponse],
) error {
	poll := time.NewTicker(vaultWatchPoll)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-lostChan:
			return nil
		case paths, open := <-changes:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.WatchVaultResponse{Paths: paths}); err != nil {
				return err
			}
		case <-poll.C:
			if now, showing := vault.GetVault(); !showing || now.ID != watched.ID {
				return stream.Send(&v1.WatchVaultResponse{IsVaultChanged: true})
			}
		}
	}
}
