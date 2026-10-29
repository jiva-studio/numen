package host

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

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

// NewTextClientFromEnv connects to the host the process was started for. It
// returns hostcall.ErrNoHost when the environment names no host.
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

// NewTextClient connects to the host at the endpoint.
func NewTextClient(endpoint hostcall.Endpoint) *TextClient {
	return NewTextClientOver(hostcall.HTTPClient(), endpoint)
}

// NewTextClientOver connects to the host at the endpoint using the given HTTP client.
func NewTextClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *TextClient {
	return &TextClient{rpc: numenv1connect.NewHostTextServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

func (c *TextClient) With(id VaultID) Texts {
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

// failText converts the error code of a response into the corresponding error.
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

func (c *TextClient) PutVersion(ctx context.Context, params TextPut) (TextVersion, error) {
	res, err := c.rpc.PutTextVersion(ctx, newTextRequest(c, &v1.PutTextVersionRequest{
		Path: params.Path, Key: params.Key, SourceVersionId: string(params.Source), Kind: string(params.Kind),
		SourceHash: params.SourceHash, File: params.File,
	}))
	if err != nil {
		return TextVersion{}, readError(err)
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return TextVersion{}, err
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
	listing := TextListing{Live: TextVersionID(res.Msg.GetLiveVersionId())}
	for _, one := range res.Msg.GetVersions() {
		listing.Versions = append(listing.Versions, readTextVersion(one))
	}
	return listing, nil
}

func (c *TextClient) ReadText(ctx context.Context, path string, id TextVersionID, cursor string) (TextPage, error) {
	res, err := c.rpc.ReadText(ctx, newTextRequest(c, &v1.ReadTextRequest{Path: path, VersionId: string(id), Cursor: cursor}))
	if err != nil {
		return TextPage{}, readError(err)
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return TextPage{}, err
	}
	return TextPage{Segments: readTextSegments(res.Msg.GetSegments()), NextCursor: res.Msg.GetNextCursor()}, nil
}

func readTextVersion(v *v1.TextVersion) TextVersion {
	return TextVersion{
		Path: v.GetPath(), PluginID: v.GetPluginId(), Key: v.GetKey(), Source: TextVersionID(v.GetSourceVersionId()),
		Kind: TextKind(v.GetKind()), File: v.GetFile(), SourceHash: v.GetSourceHash(), Hash: v.GetHash(),
		CreateTime: time.UnixMilli(v.GetCreateTimeUnixMs()),
	}
}

func readTextSegments(segments []*v1.TextSegment) []TextSegment {
	out := make([]TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, TextSegment{Text: s.GetText(), Heading: s.GetHeading(), Selector: RawSelector(s.GetSelector())})
	}
	return out
}
