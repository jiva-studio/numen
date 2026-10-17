package plugin

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoProofreaderPlugin = errors.New("no active proofreading plugin found")
)

// ProofreaderPluginHost provides plugin resolution, activation, and permission checking for proofreading.
type ProofreaderPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// ProofreaderBridge connects port.Proofreader calls to an out-of-process proofreading plugin via Connect-RPC.
type ProofreaderBridge struct {
	host     ProofreaderPluginHost
	pluginID string
	name     string
	// instruction is what the plugin is told it is proofreading.
	instruction string
}

// OpenProofreaderBridge activates or connects to the proofreading plugin and returns a port.Proofreader
// that sends instruction with every request.
func OpenProofreaderBridge(host ProofreaderPluginHost, instruction string) (*ProofreaderBridge, error) {
	plugin, ok := host.FindPluginByCapability("proofreading")
	if !ok {
		return nil, errNoProofreaderPlugin
	}

	return &ProofreaderBridge{
		host:        host,
		pluginID:    plugin.Manifest.ID,
		name:        plugin.Manifest.Name,
		instruction: instruction,
	}, nil
}

// GetName is what made a correction, recorded beside it.
func (b *ProofreaderBridge) GetName() string {
	return b.name
}

// Proofread hands over the batches and answers with what came back about each.
func (b *ProofreaderBridge) Proofread(ctx context.Context, batches []domain.ProofreadBatch) (map[int]string, error) {
	if len(batches) == 0 {
		return nil, nil
	}

	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return nil, err
		}
	}

	client, lease, err := openPlugin(ctx, b.host, b.pluginID, numenv1connect.NewProofreadingServiceClient)
	if err != nil {
		return nil, err
	}
	defer lease.Close()

	reqBatches := make([]*v1.ProofreadBatch, len(batches))
	for i, batch := range batches {
		reqBatches[i] = &v1.ProofreadBatch{
			Number: int32(batch.Number),
			Prompt: batch.Prompt,
		}
	}

	req := connect.NewRequest(&v1.ProofreadRequest{
		Batches:     reqBatches,
		Instruction: b.instruction,
	})

	resp, err := client.Proofread(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("proofread rpc: %w", err)
	}

	replies := make(map[int]string, len(resp.Msg.GetReplies()))
	for k, v := range resp.Msg.GetReplies() {
		replies[int(k)] = v
	}
	return replies, nil
}

// Close releases nothing: a proofread holds its reference on the plugin for the length of the call.
func (b *ProofreaderBridge) Close() error {
	return nil
}
