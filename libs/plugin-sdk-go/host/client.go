package host

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// Client works the vault through HostVaultService.
type Client struct {
	rpc numenv1connect.HostVaultServiceClient
	// vault is the identity of the vault the client works. Empty is the vault the window shows.
	vault string
}

// vaultHeader names the vault a call works.
const vaultHeader = "Numen-Vault"

// GetRequestVault is the vault a request a window forwards to the plugin works, as its Numen-Vault
// header names it, and empty where the request names none.
func GetRequestVault(header http.Header) VaultID {
	return VaultID(header.Get(vaultHeader))
}

func request[T any](c *Client, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if c.vault != "" {
		req.Header().Set(vaultHeader, c.vault)
	}
	return req
}

var _ Vault = (*Client)(nil)

// NewClientFromEnv connects to the host the process was started for: its endpoint and the token identifying
// the plugin come from the environment. It returns hostcall.ErrNoHost where the environment names no host.
func NewClientFromEnv() (*Client, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(addr), nil
}

// NewClientFromVars is NewClientFromEnv over the variables given.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	addr, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewClient(addr), nil
}

// NewClient connects to the host at the endpoint.
func NewClient(endpoint hostcall.Endpoint) *Client {
	return NewClientOver(hostcall.HTTPClient(), endpoint)
}

// NewClientOver connects to the host at the endpoint through the given HTTP client.
func NewClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewHostVaultServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// readError converts the failure of a call into its error: a vault the host does not know is ErrUnknownVault.
func readError(err error) error {
	if connect.CodeOf(err) == connect.CodeNotFound {
		return ErrUnknownVault
	}
	return err
}

func fail(code v1.HostVaultErrorCode) error {
	switch code {
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT:
		return ErrNoVault
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_MISSING:
		return ErrNotExist
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE:
		return ErrOutside
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_STALE:
		return ErrStale
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS:
		return ErrAlreadyExists
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_LOCKED_BY_ANOTHER:
		return ErrLockedByAnother
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE:
		return ErrNotANote
	}
	return fmt.Errorf("host: rejected with code %s", code)
}

func readFileStat(f *v1.HostFileStat) FileStat {
	if f == nil {
		return FileStat{}
	}
	return FileStat{
		Path: f.GetPath(), Kind: readKind(f), Size: f.GetSize(), ModTime: time.Unix(0, f.GetModTime()),
	}
}

// readKind returns the SourceKind of the file a file stat describes, and none for a file stat with no file.
func readKind(f *v1.HostFileStat) SourceKind {
	switch {
	case f.GetIsNote():
		return KindNote
	case f.GetPath() == "":
		return ""
	default:
		return KindFile
	}
}

func writeFileStat(f FileStat) *v1.HostFileStat {
	return &v1.HostFileStat{Path: f.Path, Size: f.Size, ModTime: f.ModTime.UnixNano(), IsNote: f.Kind == KindNote}
}

func readLink(l *v1.HostLink) ResolvedLink {
	return ResolvedLink{
		Link: Link{
			Target: Address{Scheme: l.GetTargetScheme(), Value: l.GetTargetValue()},
			Type:   l.GetLinkType(),
			Label:  l.GetLabel(),
		},
		From:        l.GetFrom(),
		To:          l.GetTo(),
		ToVault:     VaultID(l.GetToVault()),
		IsAmbiguous: l.GetIsAmbiguous(),
	}
}

func readLinks(links []*v1.HostLink) []ResolvedLink {
	out := make([]ResolvedLink, 0, len(links))
	for _, one := range links {
		out = append(out, readLink(one))
	}
	return out
}

func (c *Client) GetVault(ctx context.Context) (VaultInfo, error) {
	res, err := c.rpc.GetHostVault(ctx, request(c, &v1.GetHostVaultRequest{}))
	if err != nil {
		return VaultInfo{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return VaultInfo{}, err
	}
	return VaultInfo{ID: VaultID(res.Msg.GetId()), Name: res.Msg.GetName(), Path: res.Msg.GetPath()}, nil
}

func (c *Client) StatFile(ctx context.Context, path string) (FileStat, error) {
	res, err := c.rpc.StatFile(ctx, request(c, &v1.StatFileRequest{Path: path}))
	if err != nil {
		return FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) Read(ctx context.Context, path string) ([]byte, FileStat, error) {
	res, err := c.rpc.ReadFile(ctx, request(c, &v1.ReadFileRequest{Path: path}))
	if err != nil {
		return nil, FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, FileStat{}, err
	}
	return res.Msg.GetContent(), readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) Write(ctx context.Context, path string, content []byte, baseline FileStat) (FileStat, error) {
	res, err := c.rpc.WriteFile(ctx, request(c, &v1.WriteFileRequest{
		Path: path, Content: content, Baseline: writeFileStat(baseline),
	}))
	if err != nil {
		return FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) Create(ctx context.Context, path string, content []byte) error {
	res, err := c.rpc.CreateFile(ctx, request(c, &v1.CreateFileRequest{Path: path, Content: content}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) Lock(ctx context.Context) (func(), error) {
	res, err := c.rpc.LockVault(ctx, request(c, &v1.LockVaultRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	lock := res.Msg.GetLock()
	return func() {
		// The host lets a lock go by itself after a while, so a release that cannot reach it is not
		// a failure the caller can act on.
		_, _ = c.rpc.UnlockVault(context.WithoutCancel(ctx), request(c, &v1.UnlockVaultRequest{Lock: lock}))
	}, nil
}

func (c *Client) Scan(ctx context.Context, paths []string) error {
	res, err := c.rpc.ScanFiles(ctx, request(c, &v1.ScanFilesRequest{Paths: paths}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) GetNotes(ctx context.Context, paths []string) (map[string]NoteRef, error) {
	res, err := c.rpc.GetNoteRefs(ctx, request(c, &v1.GetNoteRefsRequest{Paths: paths}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]NoteRef, len(res.Msg.GetNotes()))
	for _, one := range res.Msg.GetNotes() {
		out[one.GetPath()] = NoteRef{Path: one.GetPath(), Title: one.GetTitle()}
	}
	return out, nil
}

func (c *Client) GetTypes(ctx context.Context, paths []string) (map[string]NoteType, error) {
	res, err := c.rpc.GetNoteTypes(ctx, request(c, &v1.GetNoteTypesRequest{Paths: paths}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]NoteType, len(res.Msg.GetTypes()))
	for path, noteType := range res.Msg.GetTypes() {
		out[path] = NoteType(noteType)
	}
	return out, nil
}

func (c *Client) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	res, err := c.rpc.GetNamedPaths(ctx, request(c, &v1.GetNamedPathsRequest{Name: name}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetPaths(), nil
}

func (c *Client) GetPathsOfType(ctx context.Context, noteType NoteType) ([]string, error) {
	res, err := c.rpc.GetPathsOfType(ctx, request(c, &v1.GetPathsOfTypeRequest{Type: string(noteType)}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetPaths(), nil
}

func (c *Client) GetLinks(ctx context.Context, from string) ([]ResolvedLink, error) {
	res, err := c.rpc.GetLinks(ctx, request(c, &v1.GetLinksRequest{Path: from}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) GetBacklinks(ctx context.Context, to string) ([]ResolvedLink, error) {
	res, err := c.rpc.GetBacklinks(ctx, request(c, &v1.GetBacklinksRequest{Path: to}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) ResolveLinks(ctx context.Context, from string, targets []string) (map[string]ResolvedLink, error) {
	res, err := c.rpc.ResolveLinks(ctx, request(c, &v1.ResolveLinksRequest{Path: from, Written: targets}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]ResolvedLink, len(res.Msg.GetLinks()))
	for key, one := range res.Msg.GetLinks() {
		out[key] = readLink(one)
	}
	return out, nil
}

func (c *Client) GetParseProblems(ctx context.Context) ([]VaultProblem, error) {
	res, err := c.rpc.GetParseProblems(ctx, request(c, &v1.GetParseProblemsRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]VaultProblem, 0, len(res.Msg.GetProblems()))
	for _, one := range res.Msg.GetProblems() {
		out = append(out, VaultProblem{Path: one.GetPath(), Kind: CheckParse, Detail: one.GetDetail()})
	}
	return out, nil
}

func (c *Client) ReadCache(ctx context.Context, name string) ([]byte, bool, error) {
	res, err := c.rpc.ReadCache(ctx, request(c, &v1.ReadCacheRequest{Name: name}))
	if err != nil {
		return nil, false, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, false, err
	}
	return res.Msg.GetContent(), res.Msg.GetFound(), nil
}

func (c *Client) WriteCache(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.WriteCache(ctx, request(c, &v1.WriteCacheRequest{Name: name, Content: content}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) Watch(ctx context.Context) (<-chan Change, error) {
	stream, err := c.rpc.WatchVault(ctx, request(c, &v1.WatchVaultRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if !stream.Receive() {
		_ = stream.Close()
		if err := stream.Err(); err != nil {
			return nil, readError(err)
		}
		return nil, fmt.Errorf("host: the watch ended before it began")
	}
	changes := make(chan Change)
	go func() {
		defer close(changes)
		defer stream.Close()
		for stream.Receive() {
			change := Change{Paths: stream.Msg().GetPaths(), IsVaultChanged: stream.Msg().GetIsVaultChanged()}
			select {
			case changes <- change:
			case <-ctx.Done():
				return
			}
		}
	}()
	return changes, nil
}

func (c *Client) With(id VaultID) Vault {
	linked := *c
	linked.vault = string(id)
	return &linked
}

func (c *Client) GetVaults(ctx context.Context) ([]VaultInfo, error) {
	res, err := c.rpc.ListHostVaults(ctx, request(c, &v1.ListHostVaultsRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]VaultInfo, 0, len(res.Msg.GetVaults()))
	for _, one := range res.Msg.GetVaults() {
		out = append(out, VaultInfo{ID: VaultID(one.GetId()), Name: one.GetName(), Path: one.GetPath()})
	}
	return out, nil
}

func (c *Client) EnsureIndexed(ctx context.Context) (IndexState, error) {
	res, err := c.rpc.EnsureVaultIndexed(ctx, request(c, &v1.EnsureVaultIndexedRequest{}))
	if err != nil {
		return IndexState{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return IndexState{}, err
	}
	out := IndexState{Reason: res.Msg.GetReason()}
	switch res.Msg.GetState() {
	case v1.HostIndexState_HOST_INDEX_STATE_INDEXED:
		out.Status = Indexed
	case v1.HostIndexState_HOST_INDEX_STATE_INDEXING:
		out.Status = Indexing
	case v1.HostIndexState_HOST_INDEX_STATE_FAILED:
		out.Status = IndexFailed
	case v1.HostIndexState_HOST_INDEX_STATE_UNSPECIFIED:
	}
	return out, nil
}

func (c *Client) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	stream, err := c.rpc.WatchMoves(ctx, request(c, &v1.WatchMovesRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if !stream.Receive() {
		_ = stream.Close()
		if err := stream.Err(); err != nil {
			return nil, readError(err)
		}
		return nil, fmt.Errorf("host: the watch ended before it began")
	}
	moves := make(chan struct{})
	go func() {
		defer close(moves)
		defer stream.Close()
		for stream.Receive() {
			select {
			case moves <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return moves, nil
}
