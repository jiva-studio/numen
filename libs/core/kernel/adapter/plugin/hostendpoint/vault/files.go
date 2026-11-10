package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// vaultLockHold is the longest a plugin holds the writers of the vault. A plugin that goes without
// releasing them must not stop the vault being written.
const vaultLockHold = 2 * time.Minute

const vaultWatchPoll = time.Second

func answer(ctx context.Context, err error) (v1.HostVaultErrorCode, error) {
	if err == nil {
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if code, ok := getErrorCode(err); ok {
		return code, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

func getErrorCode(err error) (v1.HostVaultErrorCode, bool) {
	for _, entry := range getErrorEntries() {
		if errors.Is(err, entry.err) {
			return entry.code, true
		}
	}
	return 0, false
}

func getNoteErrorCode(err error) (v1.HostVaultErrorCode, bool) {
	if code, ok := getErrorCode(err); ok {
		return code, true
	}
	noteCode, ok := wire.GetErrorCode(err)
	if !ok {
		return 0, false
	}
	code, ok := getNoteErrorCodes()[noteCode]
	return code, ok
}

// errorEntry is an error of the core and the code a vault call answers it with.
type errorEntry struct {
	err  error
	code v1.HostVaultErrorCode
}

func getErrorEntries() []errorEntry {
	return []errorEntry{
		{plugin.ErrNoVault, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT},
		{vaultdomain.ErrOutside, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE_VAULT},
		{edit.ErrStale, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT},
		{vaultdomain.ErrAlreadyExists, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS},
		{vaultdomain.ErrLockedByAnother, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_LOCKED},
		{edit.ErrNotANote, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE},
		{fs.ErrNotExist, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND},
		{plugin.ErrNotText, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT},
		{domain.ErrHierarchyCycle, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_HIERARCHY_CYCLE},
		{domain.ErrInvalidConnection, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION},
		{domain.ErrNotCreatable, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION},
		{domain.ErrNoFreeName, v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NAME_EXHAUSTED},
	}
}

func getNoteErrorCodes() map[v1.ErrorCode]v1.HostVaultErrorCode {
	return map[v1.ErrorCode]v1.HostVaultErrorCode{
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:       v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND,
		v1.ErrorCode_ERROR_CODE_NOT_A_NOTE:      v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE,
		v1.ErrorCode_ERROR_CODE_NOT_TEXT:        v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT,
		v1.ErrorCode_ERROR_CODE_TOO_LARGE:       v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_TOO_LARGE,
		v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_BODY_UNWRITABLE,
		v1.ErrorCode_ERROR_CODE_UNREADABLE:      v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNREADABLE,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS:  v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS,
		v1.ErrorCode_ERROR_CODE_INVALID_NAME:    v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_NAME,
		v1.ErrorCode_ERROR_CODE_CONFLICT:        v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT,
	}
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

func (e *filesCalls) GetHostVault(
	ctx context.Context, req *connect.Request[v1.GetHostVaultRequest],
) (*connect.Response[v1.GetHostVaultResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetHostVaultResponse{Error: noVault}), nil
	}
	v, showing := vault.Switch.GetVault()
	if !showing {
		return connect.NewResponse(&v1.GetHostVaultResponse{Error: noVault}), nil
	}
	return connect.NewResponse(&v1.GetHostVaultResponse{Path: v.Path, Id: string(v.ID), Name: v.Name}), nil
}

func (e *filesCalls) ListHostVaults(
	_ context.Context, req *connect.Request[v1.ListHostVaultsRequest],
) (*connect.Response[v1.ListHostVaultsResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListHostVaultsResponse{}), nil
	}
	recognized, err := vault.Switch.GetVaults()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	list := e.source.GetPluginVaultList()
	out := make([]*v1.HostVaultInfo, 0, len(recognized))
	for _, one := range recognized {
		out = append(out, &v1.HostVaultInfo{Id: string(one.ID), Name: one.Name, Path: one.Path, IsMissing: list != nil && list.IsMissing(one)})
	}
	return connect.NewResponse(&v1.ListHostVaultsResponse{Vaults: out}), nil
}

func (e *filesCalls) EnsureVaultIndexed(
	ctx context.Context, req *connect.Request[v1.EnsureVaultIndexedRequest],
) (*connect.Response[v1.EnsureVaultIndexedResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.EnsureVaultIndexedResponse{Error: noVault}), nil
	}
	state, err := vault.Switch.EnsureIndexed(ctx)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.EnsureVaultIndexedResponse{Error: code}), err
	}
	out := &v1.EnsureVaultIndexedResponse{Reason: state.Reason}
	switch state.Status {
	case plugin.Indexed:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_INDEXED
	case plugin.Indexing:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_INDEXING
	case plugin.IndexFailed:
		out.State = v1.HostIndexState_HOST_INDEX_STATE_FAILED
	}
	return connect.NewResponse(out), nil
}

func (e *filesCalls) StatFile(
	ctx context.Context, req *connect.Request[v1.StatFileRequest],
) (*connect.Response[v1.StatFileResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.StatFileResponse{Error: noVault}), nil
	}
	pos, err := vault.Files.StatFile(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.StatFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.StatFileResponse{Stat: writeFileStat(pos)}), nil
}

func (e *filesCalls) ReadFile(
	ctx context.Context, req *connect.Request[v1.ReadFileRequest],
) (*connect.Response[v1.ReadFileResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ReadFileResponse{Error: noVault}), nil
	}
	if req.Msg.Length != nil {
		return serveFileRange(ctx, vault.Files, req.Msg)
	}
	content, pos, err := vault.Files.Read(ctx, req.Msg.GetPath())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadFileResponse{Content: content, Stat: writeFileStat(pos)}), nil
}

func (e *filesCalls) WriteFile(
	ctx context.Context, req *connect.Request[v1.WriteFileRequest],
) (*connect.Response[v1.WriteFileResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteFileResponse{Error: noVault}), nil
	}
	pos, err := vault.Writer.WriteFile(ctx, req.Msg.GetPath(), req.Msg.GetContent(), readFileStat(req.Msg.GetBaseline()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.WriteFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.WriteFileResponse{Stat: writeFileStat(pos)}), nil
}

func (e *filesCalls) CreateFile(
	ctx context.Context, req *connect.Request[v1.CreateFileRequest],
) (*connect.Response[v1.CreateFileResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.CreateFileResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Writer.CreateFile(ctx, req.Msg.GetPath(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.CreateFileResponse{Error: code}), err
}

func (e *filesCalls) LockVault(
	ctx context.Context, req *connect.Request[v1.LockVaultRequest],
) (*connect.Response[v1.LockVaultResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.LockVaultResponse{Error: noVault}), nil
	}
	release, err := vault.Writer.Lock(ctx)
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

func (e *filesCalls) unlock(id string) {
	e.mu.Lock()
	release := e.locks[id]
	delete(e.locks, id)
	e.mu.Unlock()
	if release != nil {
		release()
	}
}

func (e *filesCalls) UnlockVault(
	_ context.Context, req *connect.Request[v1.UnlockVaultRequest],
) (*connect.Response[v1.UnlockVaultResponse], error) {
	if _, _, err := e.begin(req.Header(), manifest.PermissionVaultWrite); err != nil {
		return nil, err
	}
	e.unlock(req.Msg.GetLock())
	return connect.NewResponse(&v1.UnlockVaultResponse{}), nil
}

func (e *filesCalls) ScanFiles(
	ctx context.Context, req *connect.Request[v1.ScanFilesRequest],
) (*connect.Response[v1.ScanFilesResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ScanFilesResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Writer.Scan(ctx, req.Msg.GetPaths()))
	return connect.NewResponse(&v1.ScanFilesResponse{Error: code}), err
}

func serveFileRange(ctx context.Context, vault FileReader, req *v1.ReadFileRequest) (*connect.Response[v1.ReadFileResponse], error) {
	if req.GetStart() < 0 || req.GetLength() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid range of %s: start %d and length %d must be >= 0", req.GetPath(), req.GetStart(), req.GetLength()))
	}
	read, err := vault.ReadRange(ctx, req.GetPath(), int(req.GetStart()), int(min(req.GetLength(), math.MaxInt32)))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadFileResponse{
		Content: []byte(read.Text), Stat: writeFileStat(read.Stat), Start: int64(read.Start),
	}), nil
}
