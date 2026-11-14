package host

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// Client calls HostVaultService for the vault and HostWindowService for its window.
type Client struct {
	rpc    numenv1connect.HostVaultServiceClient
	window numenv1connect.HostWindowServiceClient
	vault  string
}

// vaultHeader names the vault a call works.
const vaultHeader = "Numen-Vault"

// GetRequestVault is the vault a request a window forwards to the plugin works, as its Numen-Vault
// header names it, and empty where the request names none.
func GetRequestVault(header http.Header) domain.VaultID {
	return domain.VaultID(header.Get(vaultHeader))
}

func request[T any](c *Client, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if c.vault != "" {
		req.Header().Set(vaultHeader, c.vault)
	}
	return req
}

var (
	_ VaultSelector = (*Client)(nil)
	_ VaultIndex    = (*Client)(nil)
	_ FileReader    = (*Client)(nil)
	_ FileWriter    = (*Client)(nil)
	_ NoteLookup    = (*Client)(nil)
	_ LinkLookup    = (*Client)(nil)
	_ CacheStore    = (*Client)(nil)
)

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

func NewClient(endpoint hostcall.Endpoint) *Client {
	return NewClientOver(hostcall.HTTPClient(), endpoint)
}

func NewClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{
		rpc:    numenv1connect.NewHostVaultServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...),
		window: numenv1connect.NewHostWindowServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...),
	}
}

func readError(err error) error {
	if connect.CodeOf(err) == connect.CodeNotFound {
		return ErrUnknownVault
	}
	return err
}

func getHostVaultErrors() map[v1.HostVaultErrorCode]error {
	return map[v1.HostVaultErrorCode]error{
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT:       ErrNoVault,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND:      ErrNotExist,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE_VAULT:  ErrOutside,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT:       ErrStale,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS: ErrAlreadyExists,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_LOCKED:         ErrLockedByAnother,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE:     ErrNotANote,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_NAME:   domain.ErrInvalidName,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT:       ErrNotText,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED:  ErrNotAttempted,
	}
}

func fail(code v1.HostVaultErrorCode) error {
	if code == v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED {
		return nil
	}
	if err, isKnown := getHostVaultErrors()[code]; isKnown {
		return err
	}
	return fmt.Errorf("the host returned the error code %s", code)
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

func writeFileStat(f domain.FileStat) *v1.HostFileStat {
	if f.IsZero() {
		return nil
	}
	return &v1.HostFileStat{Path: f.Path, Size: f.Size, ModTime: f.ModTime.UnixNano(), IsNote: f.Kind == domain.KindNote}
}

func readLink(l *v1.HostLink) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{
			Target: domain.Address{Scheme: l.GetTargetScheme(), Value: l.GetTargetValue()},
			Role:   readRole(l.GetRole()),
			Type:   l.GetLinkType(),
			Why:    l.GetNote(),
			Label:  l.GetLabel(),
			Marker: readMarker(l.GetMarker()),
		},
		From:        l.GetFrom(),
		To:          l.GetTo(),
		ToVault:     domain.VaultID(l.GetToVault()),
		IsAmbiguous: l.GetIsAmbiguous(),
	}
}

func readLinks(links []*v1.HostLink) []domain.ResolvedLink {
	out := make([]domain.ResolvedLink, 0, len(links))
	for _, one := range links {
		out = append(out, readLink(one))
	}
	return out
}

func (c *Client) GetVault(ctx context.Context) (domain.Vault, error) {
	res, err := c.rpc.GetHostVault(ctx, request(c, &v1.GetHostVaultRequest{}))
	if err != nil {
		return domain.Vault{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.Vault{}, err
	}
	return domain.Vault{ID: domain.VaultID(res.Msg.GetId()), Name: res.Msg.GetName(), Path: res.Msg.GetPath()}, nil
}

func (c *Client) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	res, err := c.rpc.StatFile(ctx, request(c, &v1.StatFileRequest{Path: path}))
	if err != nil {
		return domain.FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	res, err := c.rpc.ReadFile(ctx, request(c, &v1.ReadFileRequest{Path: path}))
	if err != nil {
		return nil, domain.FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, domain.FileStat{}, err
	}
	return res.Msg.GetContent(), readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) ReadRange(ctx context.Context, path string, start, length int) (FileRange, error) {
	size := int64(length)
	res, err := c.rpc.ReadFile(ctx, request(c, &v1.ReadFileRequest{Path: path, Start: int64(start), Length: &size}))
	if err != nil {
		return FileRange{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return FileRange{}, err
	}
	return FileRange{Text: string(res.Msg.GetContent()), Start: int(res.Msg.GetStart()), Stat: readFileStat(res.Msg.GetStat())}, nil
}

func (c *Client) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	res, err := c.rpc.WriteFile(ctx, request(c, &v1.WriteFileRequest{
		Path: path, Content: content, Baseline: writeFileStat(baseline),
	}))
	if err != nil {
		return domain.FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.FileStat{}, err
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

func (c *Client) Lock(ctx context.Context) (func() error, error) {
	res, err := c.rpc.LockVault(ctx, request(c, &v1.LockVaultRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	lock := res.Msg.GetLock()
	return func() error {
		if _, err := c.rpc.UnlockVault(context.WithoutCancel(ctx), request(c, &v1.UnlockVaultRequest{Lock: lock})); err != nil {
			return fmt.Errorf("release vault lock: %w", err)
		}
		return nil
	}, nil
}

func (c *Client) Scan(ctx context.Context, paths []string) error {
	res, err := c.rpc.ScanFiles(ctx, request(c, &v1.ScanFilesRequest{Paths: paths}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	res, err := c.rpc.GetNoteRefs(ctx, request(c, &v1.GetNoteRefsRequest{Paths: paths}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]domain.NoteRef, len(res.Msg.GetNotes()))
	for _, one := range res.Msg.GetNotes() {
		out[one.GetPath()] = domain.NoteRef{Path: one.GetPath(), Title: one.GetTitle(), ID: one.GetId()}
	}
	return out, nil
}

func (c *Client) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	res, err := c.rpc.GetNoteTypes(ctx, request(c, &v1.GetNoteTypesRequest{Paths: paths}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]domain.NoteType, len(res.Msg.GetTypes()))
	for path, noteType := range res.Msg.GetTypes() {
		out[path] = domain.NoteType(noteType)
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

func (c *Client) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	res, err := c.rpc.GetPathsOfType(ctx, request(c, &v1.GetPathsOfTypeRequest{Type: string(noteType)}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetPaths(), nil
}

func (c *Client) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	res, err := c.rpc.GetLinks(ctx, request(c, &v1.GetLinksRequest{Path: from}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	res, err := c.rpc.GetBacklinks(ctx, request(c, &v1.GetBacklinksRequest{Path: to}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) ResolveLinks(ctx context.Context, from string, targets []string) (map[string]domain.ResolvedLink, error) {
	res, err := c.rpc.ResolveLinks(ctx, request(c, &v1.ResolveLinksRequest{Path: from, Addresses: targets}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]domain.ResolvedLink, len(res.Msg.GetLinks()))
	for key, one := range res.Msg.GetLinks() {
		out[key] = readLink(one)
	}
	return out, nil
}

func (c *Client) GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error) {
	res, err := c.rpc.GetParseProblems(ctx, request(c, &v1.GetParseProblemsRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]domain.VaultProblem, 0, len(res.Msg.GetProblems()))
	for _, one := range res.Msg.GetProblems() {
		out = append(out, domain.VaultProblem{Path: one.GetPath(), Kind: domain.CheckParse, Detail: one.GetDetail()})
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
	return res.Msg.GetContent(), res.Msg.GetIsFound(), nil
}

func (c *Client) WriteCache(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.WriteCache(ctx, request(c, &v1.WriteCacheRequest{Name: name, Content: content}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

// With returns the client for the vault with the given identity, used as the shown vault is. A call on
// it returns ErrUnknownVault when the installation has no such vault.
func (c *Client) With(id domain.VaultID) *Client {
	linked := *c
	linked.vault = string(id)
	return &linked
}

func (c *Client) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	res, err := c.rpc.ListHostVaults(ctx, request(c, &v1.ListHostVaultsRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]domain.Vault, 0, len(res.Msg.GetVaults()))
	for _, one := range res.Msg.GetVaults() {
		out = append(out, domain.Vault{ID: domain.VaultID(one.GetId()), Name: one.GetName(), Path: one.GetPath()})
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

func (c *Client) WatchMoves(ctx context.Context) (<-chan Move, error) {
	stream, err := c.rpc.WatchMoves(ctx, request(c, &v1.WatchMovesRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if !stream.Receive() {
		closeErr := stream.Close()
		if err := stream.Err(); err != nil {
			return nil, errors.Join(readError(err), closeErr)
		}
		return nil, errors.Join(fmt.Errorf("host: the watch ended before it began"), closeErr)
	}
	moves := make(chan Move)
	go followStream(ctx, stream, moves, func(*v1.WatchMovesResponse) Move { return Move{Err: nil} },
		func(err error) Move { return Move{Err: err} })
	return moves, nil
}
