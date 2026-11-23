// Package searchfunctiontest calls the query functions of a plugin as the host does, for the tests of the
// plugin.
package searchfunctiontest

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Client calls the functions a plugin serves at a base URL.
type Client struct {
	rpc numenv1connect.SearchFunctionServiceClient
}

// NewClient returns a client that presents token on every call, as the host does.
func NewClient(httpClient *http.Client, baseURL, token string) *Client {
	return &Client{rpc: numenv1connect.NewSearchFunctionServiceClient(httpClient, baseURL, connect.WithInterceptors(transport.NewBearer(token)))}
}

// Call is one call of a function: its name, the vault, the generation of the index and the arguments by name,
// each a string, an int64 or a bool.
type Call struct {
	Function        string
	VaultID         string
	IndexGeneration string
	Arguments       map[string]any
}

func (c Call) write() *v1.SearchFunctionCall {
	wire := &v1.SearchFunctionCall{Function: c.Function, VaultId: c.VaultID, IndexGeneration: c.IndexGeneration}
	for name, value := range c.Arguments {
		one := &v1.SearchFunctionArgument{Name: name}
		switch typed := value.(type) {
		case string:
			one.Value = &v1.SearchFunctionArgument_StringValue{StringValue: typed}
		case int64:
			one.Value = &v1.SearchFunctionArgument_IntValue{IntValue: typed}
		case bool:
			one.Value = &v1.SearchFunctionArgument_BoolValue{BoolValue: typed}
		}
		wire.Arguments = append(wire.Arguments, one)
	}
	return wire
}

// MatchDocuments returns the paths among paths the call holds for, and the warning of the answer.
func (c *Client) MatchDocuments(ctx context.Context, call Call, paths []string) ([]string, string, error) {
	res, err := c.rpc.MatchDocuments(ctx, connect.NewRequest(&v1.MatchDocumentsRequest{Call: call.write(), Paths: paths}))
	if err != nil {
		return nil, "", err
	}
	return res.Msg.GetPaths(), res.Msg.GetWarning(), nil
}

// FindDocuments returns at most limit paths the call holds for, their estimated count and the warning.
func (c *Client) FindDocuments(ctx context.Context, call Call, limit int32) ([]string, int64, string, error) {
	res, err := c.rpc.FindDocuments(ctx, connect.NewRequest(&v1.FindDocumentsRequest{Call: call.write(), Limit: limit}))
	if err != nil {
		return nil, 0, "", err
	}
	return res.Msg.GetPaths(), res.Msg.GetEstimatedCount(), res.Msg.GetWarning(), nil
}
