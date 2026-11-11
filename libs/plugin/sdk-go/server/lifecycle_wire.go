package server

import (
	"context"
	"maps"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// lifecycleHandler serves a LifecycleServer as numen.v1.PluginLifecycleService.
type lifecycleHandler struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler

	state *LifecycleServer
}

func (h lifecycleHandler) Ping(
	context.Context, *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(writePing(PingResult{Status: statusOK, Protocol: ProtocolVersion})), nil
}

func (h lifecycleHandler) Shutdown(
	context.Context, *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	if err := h.state.shutdown(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ShutdownResponse{Success: true}), nil
}

func (h lifecycleHandler) Configure(
	ctx context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	if err := h.state.configureWith(ctx, req.Msg.GetValues(), readSecrets(req.Msg.GetSecrets())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

// PingResult is the answer of a plugin to a ping.
//
//numen:api
type PingResult struct {
	// Status is "ok" for a healthy plugin.
	Status string
	// Protocol is the protocol version the plugin uses.
	Protocol int32
}

func readPing(res *v1.PingResponse) PingResult {
	return PingResult{Status: res.GetStatus(), Protocol: res.GetProtocol()}
}

func writePing(result PingResult) *v1.PingResponse {
	return &v1.PingResponse{Status: result.Status, Protocol: result.Protocol}
}

// LifecycleClient calls the lifecycle service of a plugin.
type LifecycleClient struct {
	rpc numenv1connect.PluginLifecycleServiceClient
}

func NewLifecycleClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) *LifecycleClient {
	return &LifecycleClient{rpc: numenv1connect.NewPluginLifecycleServiceClient(httpClient, baseURL, opts...)}
}

// Ping asks the plugin for its status and protocol version.
func (c *LifecycleClient) Ping(ctx context.Context) (PingResult, error) {
	res, err := c.rpc.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		return PingResult{}, err
	}
	return readPing(res.Msg), nil
}

// Configure sends the configuration values and the secrets to the plugin and reports whether it accepted them.
func (c *LifecycleClient) Configure(ctx context.Context, values map[string]string, secrets Secrets) (bool, error) {
	res, err := c.rpc.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{Values: values, Secrets: writeSecrets(secrets)}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}

// Shutdown asks the plugin to terminate and reports whether it accepted.
func (c *LifecycleClient) Shutdown(ctx context.Context) (bool, error) {
	res, err := c.rpc.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}

func readSecrets(entries []*v1.ConfigureSecrets) Secrets {
	if len(entries) == 0 {
		return nil
	}
	secrets := make(Secrets, len(entries))
	for _, entry := range entries {
		list := make([]Secret, 0, len(entry.GetValues()))
		for _, one := range entry.GetValues() {
			list = append(list, Secret{ID: one.GetId(), Name: one.GetName(), Group: one.GetGroup(), Value: one.GetValue()})
		}
		secrets[entry.GetName()] = list
	}
	return secrets
}

func writeSecrets(secrets Secrets) []*v1.ConfigureSecrets {
	entries := make([]*v1.ConfigureSecrets, 0, len(secrets))
	for _, name := range slices.Sorted(maps.Keys(secrets)) {
		values := make([]*v1.SecretValue, 0, len(secrets[name]))
		for _, one := range secrets[name] {
			values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group, Value: one.Value})
		}
		entries = append(entries, &v1.ConfigureSecrets{Name: name, Values: values})
	}
	return entries
}
