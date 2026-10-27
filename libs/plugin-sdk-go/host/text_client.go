package host

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// TextClient works the text versions of a vault through HostTextService.
type TextClient struct {
	rpc numenv1connect.HostTextServiceClient
	// vault is the identity of the vault the client works. Empty is the vault the window shows.
	vault string
}

var _ Texts = (*TextClient)(nil)

// NewTextClientFromEnv reaches the host the process was started for. It returns hostcall.ErrNoHost
// where the host named none.
func NewTextClientFromEnv() (*TextClient, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewTextClient(addr), nil
}

// NewTextClientFromVars is NewTextClientFromEnv over the variables given.
func NewTextClientFromVars(vars map[string]string) (*TextClient, error) {
	addr, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewTextClient(addr), nil
}

// NewTextClient reaches the host at the endpoint.
func NewTextClient(endpoint hostcall.Endpoint) *TextClient {
	return NewTextClientOver(hostcall.HTTPClient(), endpoint)
}

// NewTextClientOver reaches the host at the endpoint through the HTTP client given.
func NewTextClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *TextClient {
	return &TextClient{rpc: numenv1connect.NewHostTextServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

func (c *TextClient) With(id domain.VaultID) Texts {
	linked := *c
	linked.vault = string(id)
	return &linked
}

func newTextRequest[T any](c *TextClient, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if c.vault != "" {
		req.Header().Set(vaultHeader, c.vault)
	}
	return req
}

// failText turns the code of an answer into the error it stands for.
func failText(code v1.TextErrorCode) error {
	switch code {
	case v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.TextErrorCode_TEXT_ERROR_CODE_MISSING:
		return ErrNotExist
	case v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT:
		return ErrNoVault
	case v1.TextErrorCode_TEXT_ERROR_CODE_INVALID:
		return ErrInvalidVersion
	case v1.TextErrorCode_TEXT_ERROR_CODE_NOT_OWNER:
		return ErrNotOwner
	}
	return fmt.Errorf("host: rejected with code %s", code)
}

func (c *TextClient) PutVersion(ctx context.Context, params TextPut) (domain.TextVersion, error) {
	res, err := c.rpc.PutTextVersion(ctx, newTextRequest(c, &v1.PutTextVersionRequest{
		Path: params.Path, Key: params.Key, SourceVersionId: string(params.Source), Kind: string(params.Kind),
		SourceHash: params.SourceHash, File: params.File,
	}))
	if err != nil {
		return domain.TextVersion{}, readError(err)
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return domain.TextVersion{}, err
	}
	return readTextVersion(res.Msg.GetVersion()), nil
}

func (c *TextClient) RemoveVersion(ctx context.Context, path, key string) error {
	res, err := c.rpc.RemoveTextVersion(ctx, newTextRequest(c, &v1.RemoveTextVersionRequest{Path: path, Key: key}))
	if err != nil {
		return readError(err)
	}
	return failText(res.Msg.GetError())
}

func (c *TextClient) ListVersions(ctx context.Context, path string) (TextListing, error) {
	res, err := c.rpc.ListTextVersions(ctx, newTextRequest(c, &v1.ListTextVersionsRequest{Path: path}))
	if err != nil {
		return TextListing{}, readError(err)
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return TextListing{}, err
	}
	listing := TextListing{Live: domain.TextVersionID(res.Msg.GetLiveVersionId())}
	for _, one := range res.Msg.GetVersions() {
		listing.Versions = append(listing.Versions, readTextVersion(one))
	}
	return listing, nil
}

func (c *TextClient) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (TextPage, error) {
	res, err := c.rpc.ReadText(ctx, newTextRequest(c, &v1.ReadTextRequest{Path: path, VersionId: string(id), Cursor: cursor}))
	if err != nil {
		return TextPage{}, readError(err)
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return TextPage{}, err
	}
	return TextPage{Segments: readTextSegments(res.Msg.GetSegments()), NextCursor: res.Msg.GetNextCursor()}, nil
}

func readTextVersion(v *v1.TextVersion) domain.TextVersion {
	return domain.TextVersion{
		Path: v.GetPath(), PluginID: v.GetPluginId(), Key: v.GetKey(), Source: domain.TextVersionID(v.GetSourceVersionId()),
		Kind: domain.TextKind(v.GetKind()), File: v.GetFile(), SourceHash: v.GetSourceHash(), Hash: v.GetHash(),
		CreateTime: time.UnixMilli(v.GetCreateTimeUnixMs()),
	}
}

func readTextSegments(segments []*v1.TextSegment) []domain.TextSegment {
	out := make([]domain.TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, domain.TextSegment{Text: s.GetText(), Heading: s.GetHeading(), Selector: domain.Selector(s.GetSelector())})
	}
	return out
}
