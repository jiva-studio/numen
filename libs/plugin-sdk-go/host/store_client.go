package host

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// StoreClient works the plugin's folder through HostStoreService.
type StoreClient struct {
	rpc numenv1connect.HostStoreServiceClient
	// vault is the identity of the vault the client works. Empty is the vault the window shows.
	vault string
}

var _ Store = (*StoreClient)(nil)

// NewStoreClientFromEnv reaches the host the process was started for. It returns hostcall.ErrNoHost
// where the host named none.
func NewStoreClientFromEnv() (*StoreClient, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewStoreClient(addr), nil
}

// NewStoreClientFromVars is NewStoreClientFromEnv over the variables given.
func NewStoreClientFromVars(vars map[string]string) (*StoreClient, error) {
	addr, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewStoreClient(addr), nil
}

// NewStoreClient reaches the host at the endpoint.
func NewStoreClient(endpoint hostcall.Endpoint) *StoreClient {
	return NewStoreClientOver(hostcall.HTTPClient(), endpoint)
}

// NewStoreClientOver reaches the host at the endpoint through the HTTP client given.
func NewStoreClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *StoreClient {
	return &StoreClient{rpc: numenv1connect.NewHostStoreServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

func (c *StoreClient) With(id VaultID) Store {
	linked := *c
	linked.vault = string(id)
	return &linked
}

func newStoreRequest[T any](c *StoreClient, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if c.vault != "" {
		req.Header().Set(vaultHeader, c.vault)
	}
	return req
}

func readStoreError(code v1.HostStoreErrorCode) error {
	switch code {
	case v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_NO_VAULT:
		return ErrNoVault
	case v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING:
		return ErrNotExist
	case v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_OUTSIDE:
		return ErrOutside
	case v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_LOCKED_BY_ANOTHER:
		return ErrLockedByAnother
	}
	return fmt.Errorf("host: rejected with code %s", code)
}

func (c *StoreClient) Read(ctx context.Context, name string) ([]byte, error) {
	res, err := c.rpc.ReadStoreFile(ctx, newStoreRequest(c, &v1.ReadStoreFileRequest{Name: name}))
	if err != nil {
		return nil, readError(err)
	}
	if err := readStoreError(res.Msg.GetError()); err != nil {
		return nil, err
	}
	return res.Msg.GetContent(), nil
}

func (c *StoreClient) Append(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.AppendStoreFile(ctx, newStoreRequest(c, &v1.AppendStoreFileRequest{Name: name, Content: content}))
	if err != nil {
		return readError(err)
	}
	return readStoreError(res.Msg.GetError())
}

func (c *StoreClient) List(ctx context.Context) ([]StoreFile, error) {
	res, err := c.rpc.ListStoreFiles(ctx, newStoreRequest(c, &v1.ListStoreFilesRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := readStoreError(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]StoreFile, 0, len(res.Msg.GetFiles()))
	for _, one := range res.Msg.GetFiles() {
		out = append(out, StoreFile{Name: one.GetName(), Size: one.GetSize()})
	}
	return out, nil
}

func (c *StoreClient) Write(ctx context.Context, name string, content []byte) error {
	res, err := c.rpc.WriteStoreFile(ctx, newStoreRequest(c, &v1.WriteStoreFileRequest{Name: name, Content: content}))
	if err != nil {
		return readError(err)
	}
	return readStoreError(res.Msg.GetError())
}

func (c *StoreClient) Delete(ctx context.Context, name string) error {
	res, err := c.rpc.DeleteStoreFile(ctx, newStoreRequest(c, &v1.DeleteStoreFileRequest{Name: name}))
	if err != nil {
		return readError(err)
	}
	return readStoreError(res.Msg.GetError())
}

func (c *StoreClient) Rename(ctx context.Context, from, to string) error {
	res, err := c.rpc.RenameStoreFile(ctx, newStoreRequest(c, &v1.RenameStoreFileRequest{From: from, To: to}))
	if err != nil {
		return readError(err)
	}
	return readStoreError(res.Msg.GetError())
}
