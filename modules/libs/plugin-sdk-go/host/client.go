package host

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Client works the vault through HostVaultService.
type Client struct {
	rpc numenv1connect.HostVaultServiceClient
}

var _ Vault = (*Client)(nil)

// NewClientFromEnv reaches the host the process was started for: its endpoint and the token it knows
// the plugin by come from the environment. It returns hostcall.ErrNoHost where the host named none.
func NewClientFromEnv() (*Client, error) {
	at, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(at), nil
}

// NewClientFromVars is NewClientFromEnv over the variables given.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	at, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewClient(at), nil
}

// NewClient reaches the host at the endpoint.
func NewClient(at hostcall.Endpoint) *Client {
	return NewClientOver(hostcall.HTTPClient(), at)
}

// NewClientOver reaches the host at the endpoint through the HTTP client given.
func NewClientOver(httpClient connect.HTTPClient, at hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewHostVaultServiceClient(httpClient, at.BaseURL, at.Options()...)}
}

// fail turns the code of an answer into the error it stands for.
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
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OCCUPIED:
		return ErrOccupied
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_HELD_BY_ANOTHER:
		return ErrHeldByAnother
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_DATA_FOLDER:
		return ErrNoDataFolder
	case v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE:
		return ErrNotANote
	}
	return fmt.Errorf("host: refused with code %s", code)
}

func readFingerprint(f *v1.HostFingerprint) domain.Fingerprint {
	if f == nil {
		return domain.Fingerprint{}
	}
	return domain.Fingerprint{Path: f.GetPath(), Size: f.GetSize(), ModTime: time.Unix(0, f.GetModTime())}
}

func writeFingerprint(f domain.Fingerprint) *v1.HostFingerprint {
	return &v1.HostFingerprint{Path: f.Path, Size: f.Size, ModTime: f.ModTime.UnixNano()}
}

func readLink(l *v1.HostLink) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{
			Target: domain.Address{Scheme: l.GetTargetScheme(), Value: l.GetTargetValue()},
			Type:   l.GetLinkType(),
			Label:  l.GetLabel(),
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
	res, err := c.rpc.GetHostVault(ctx, connect.NewRequest(&v1.GetHostVaultRequest{}))
	if err != nil {
		return domain.Vault{}, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.Vault{}, err
	}
	return domain.Vault{ID: domain.VaultID(res.Msg.GetId()), Name: res.Msg.GetName(), Path: res.Msg.GetPath()}, nil
}

func (c *Client) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	res, err := c.rpc.GetFileFingerprint(ctx, connect.NewRequest(&v1.GetFileFingerprintRequest{Path: path}))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.Fingerprint{}, err
	}
	return readFingerprint(res.Msg.GetFingerprint()), nil
}

func (c *Client) Read(ctx context.Context, path string) ([]byte, domain.Fingerprint, error) {
	res, err := c.rpc.ReadFile(ctx, connect.NewRequest(&v1.ReadFileRequest{Path: path}))
	if err != nil {
		return nil, domain.Fingerprint{}, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, domain.Fingerprint{}, err
	}
	return res.Msg.GetContent(), readFingerprint(res.Msg.GetFingerprint()), nil
}

func (c *Client) Write(ctx context.Context, path string, content []byte, baseline domain.Fingerprint) (domain.Fingerprint, error) {
	res, err := c.rpc.WriteFile(ctx, connect.NewRequest(&v1.WriteFileRequest{
		Path: path, Content: content, Baseline: writeFingerprint(baseline),
	}))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.Fingerprint{}, err
	}
	return readFingerprint(res.Msg.GetFingerprint()), nil
}

func (c *Client) Create(ctx context.Context, path string, content []byte) error {
	res, err := c.rpc.CreateFile(ctx, connect.NewRequest(&v1.CreateFileRequest{Path: path, Content: content}))
	if err != nil {
		return err
	}
	return fail(res.Msg.GetError())
}

func (c *Client) Lock(ctx context.Context) (func(), error) {
	res, err := c.rpc.LockVault(ctx, connect.NewRequest(&v1.LockVaultRequest{}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	lock := res.Msg.GetLock()
	return func() {
		// The host lets a lock go by itself after a while, so a release that cannot reach it is not
		// a failure the caller can act on.
		_, _ = c.rpc.UnlockVault(context.WithoutCancel(ctx), connect.NewRequest(&v1.UnlockVaultRequest{Lock: lock}))
	}, nil
}

func (c *Client) Scan(ctx context.Context, paths []string) error {
	res, err := c.rpc.ScanFiles(ctx, connect.NewRequest(&v1.ScanFilesRequest{Paths: paths}))
	if err != nil {
		return err
	}
	return fail(res.Msg.GetError())
}

func (c *Client) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	res, err := c.rpc.GetNoteRefs(ctx, connect.NewRequest(&v1.GetNoteRefsRequest{Paths: paths}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make(map[string]domain.NoteRef, len(res.Msg.GetNotes()))
	for _, one := range res.Msg.GetNotes() {
		out[one.GetPath()] = domain.NoteRef{Path: one.GetPath(), Title: one.GetTitle()}
	}
	return out, nil
}

func (c *Client) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	res, err := c.rpc.GetNoteTypes(ctx, connect.NewRequest(&v1.GetNoteTypesRequest{Paths: paths}))
	if err != nil {
		return nil, err
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

func (c *Client) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	res, err := c.rpc.GetPathsOfType(ctx, connect.NewRequest(&v1.GetPathsOfTypeRequest{Type: string(noteType)}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetPaths(), nil
}

func (c *Client) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	res, err := c.rpc.GetLinks(ctx, connect.NewRequest(&v1.GetLinksRequest{Path: from}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	res, err := c.rpc.GetBacklinks(ctx, connect.NewRequest(&v1.GetBacklinksRequest{Path: to}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return readLinks(res.Msg.GetLinks()), nil
}

func (c *Client) ResolveLinks(ctx context.Context, from string, written []string) (map[string]domain.ResolvedLink, error) {
	res, err := c.rpc.ResolveLinks(ctx, connect.NewRequest(&v1.ResolveLinksRequest{Path: from, Written: written}))
	if err != nil {
		return nil, err
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
	res, err := c.rpc.GetParseProblems(ctx, connect.NewRequest(&v1.GetParseProblemsRequest{}))
	if err != nil {
		return nil, err
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

func (c *Client) ReadData(ctx context.Context, name string) ([]byte, error) {
	res, err := c.rpc.ReadData(ctx, connect.NewRequest(&v1.ReadDataRequest{Name: name}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetContent(), nil
}

func (c *Client) AppendData(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.AppendData(ctx, connect.NewRequest(&v1.AppendDataRequest{Name: name, Content: content}))
	if err != nil {
		return err
	}
	return fail(res.Msg.GetError())
}

func (c *Client) ListData(ctx context.Context) ([]DataFile, error) {
	res, err := c.rpc.ListData(ctx, connect.NewRequest(&v1.ListDataRequest{}))
	if err != nil {
		return nil, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]DataFile, 0, len(res.Msg.GetFiles()))
	for _, one := range res.Msg.GetFiles() {
		out = append(out, DataFile{Name: one.GetName(), Size: one.GetSize()})
	}
	return out, nil
}

func (c *Client) ReadCache(ctx context.Context, name string) ([]byte, bool, error) {
	res, err := c.rpc.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: name}))
	if err != nil {
		return nil, false, err
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, false, err
	}
	return res.Msg.GetContent(), res.Msg.GetFound(), nil
}

func (c *Client) WriteCache(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.WriteCache(ctx, connect.NewRequest(&v1.WriteCacheRequest{Name: name, Content: content}))
	if err != nil {
		return err
	}
	return fail(res.Msg.GetError())
}

func (c *Client) Watch(ctx context.Context) (<-chan Change, error) {
	stream, err := c.rpc.WatchVault(ctx, connect.NewRequest(&v1.WatchVaultRequest{}))
	if err != nil {
		return nil, err
	}
	if !stream.Receive() {
		_ = stream.Close()
		if err := stream.Err(); err != nil {
			return nil, err
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
