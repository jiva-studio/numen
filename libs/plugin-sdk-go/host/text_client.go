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

// TextClient works the text versions of the vault the host shows through HostTextService.
type TextClient struct {
	rpc numenv1connect.HostTextServiceClient
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

// NewTextClient reaches the host at the endpoint.
func NewTextClient(endpoint hostcall.Endpoint) *TextClient {
	return NewTextClientOver(hostcall.HTTPClient(), endpoint)
}

// NewTextClientOver reaches the host at the endpoint through the HTTP client given.
func NewTextClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *TextClient {
	return &TextClient{rpc: numenv1connect.NewHostTextServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
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

func (c *TextClient) PutVersion(ctx context.Context, put TextPut) (domain.TextVersion, error) {
	res, err := c.rpc.PutTextVersion(ctx, connect.NewRequest(&v1.PutTextVersionRequest{
		Path: put.Path, Key: put.Key, InputVersionId: string(put.Input), Format: writeTextFormat(put.Format),
		InputHash: put.InputHash, Hash: put.Hash,
	}))
	if err != nil {
		return domain.TextVersion{}, err
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return domain.TextVersion{}, err
	}
	return readTextVersion(res.Msg.GetVersion()), nil
}

func (c *TextClient) RemoveVersion(ctx context.Context, path, key string) error {
	res, err := c.rpc.RemoveTextVersion(ctx, connect.NewRequest(&v1.RemoveTextVersionRequest{Path: path, Key: key}))
	if err != nil {
		return err
	}
	return failText(res.Msg.GetError())
}

func (c *TextClient) ListVersions(ctx context.Context, path string) (TextListing, error) {
	res, err := c.rpc.ListTextVersions(ctx, connect.NewRequest(&v1.ListTextVersionsRequest{Path: path}))
	if err != nil {
		return TextListing{}, err
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
	res, err := c.rpc.ReadText(ctx, connect.NewRequest(&v1.ReadTextRequest{Path: path, VersionId: string(id), Cursor: cursor}))
	if err != nil {
		return TextPage{}, err
	}
	if err := failText(res.Msg.GetError()); err != nil {
		return TextPage{}, err
	}
	return TextPage{Parts: readTextParts(res.Msg.GetParts()), NextCursor: res.Msg.GetNextCursor()}, nil
}

func readTextVersion(v *v1.TextVersion) domain.TextVersion {
	return domain.TextVersion{
		Path: v.GetPath(), PluginID: v.GetPluginId(), Key: v.GetKey(), Input: domain.TextVersionID(v.GetInputVersionId()),
		Format: readTextFormat(v.GetFormat()), InputHash: v.GetInputHash(), Hash: v.GetHash(),
		CreateTime: time.UnixMilli(v.GetCreateTimeUnixMs()),
	}
}

func readTextFormat(f v1.TextFormat) domain.TextFormat {
	switch f {
	case v1.TextFormat_TEXT_FORMAT_PLAIN:
		return domain.TextFormatPlain
	case v1.TextFormat_TEXT_FORMAT_TIMED:
		return domain.TextFormatTimed
	case v1.TextFormat_TEXT_FORMAT_POSITIONED:
		return domain.TextFormatPositioned
	case v1.TextFormat_TEXT_FORMAT_UNSPECIFIED:
	}
	return ""
}

func writeTextFormat(f domain.TextFormat) v1.TextFormat {
	switch f {
	case domain.TextFormatPlain:
		return v1.TextFormat_TEXT_FORMAT_PLAIN
	case domain.TextFormatTimed:
		return v1.TextFormat_TEXT_FORMAT_TIMED
	case domain.TextFormatPositioned:
		return v1.TextFormat_TEXT_FORMAT_POSITIONED
	}
	return v1.TextFormat_TEXT_FORMAT_UNSPECIFIED
}

func readTextParts(parts []*v1.TextPart) []domain.TextPart {
	out := make([]domain.TextPart, 0, len(parts))
	for _, p := range parts {
		place := p.GetPlace()
		rects := make([]domain.Rect, 0, len(place.GetRects()))
		for _, r := range place.GetRects() {
			rects = append(rects, domain.Rect{MinX: r.GetMinX(), MinY: r.GetMinY(), MaxX: r.GetMaxX(), MaxY: r.GetMaxY()})
		}
		out = append(out, domain.TextPart{
			Text: p.GetText(), Section: p.GetSection(),
			Place: domain.TextPlace{
				Page: int(place.GetPage()), Rects: rects,
				Start: time.Duration(place.GetStartMs()) * time.Millisecond,
				End:   time.Duration(place.GetEndMs()) * time.Millisecond,
			},
		})
	}
	return out
}

func writeTextParts(parts []domain.TextPart) []*v1.TextPart {
	out := make([]*v1.TextPart, 0, len(parts))
	for _, p := range parts {
		rects := make([]*v1.TextRect, 0, len(p.Place.Rects))
		for _, r := range p.Place.Rects {
			rects = append(rects, &v1.TextRect{MinX: r.MinX, MinY: r.MinY, MaxX: r.MaxX, MaxY: r.MaxY})
		}
		out = append(out, &v1.TextPart{
			Text: p.Text, Section: p.Section,
			Place: &v1.TextPlace{
				Page: int32(p.Place.Page), Rects: rects,
				StartMs: p.Place.Start.Milliseconds(), EndMs: p.Place.End.Milliseconds(),
			},
		})
	}
	return out
}
