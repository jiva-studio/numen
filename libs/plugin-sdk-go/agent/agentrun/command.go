package agentrun

import (
	"context"
	"strings"
	"sync"

	"connectrpc.com/connect"
)

// ExecutablePath holds the path of the program an agent plugin runs, as the host configures it.
// The zero value is ready.
type ExecutablePath struct {
	mu   sync.RWMutex
	path string
}

// ApplyConfig keeps the executable_path value of the configuration, trimmed. A configuration that
// does not name it leaves the path as it is.
func (e *ExecutablePath) ApplyConfig(values map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if path, ok := values["executable_path"]; ok {
		e.path = strings.TrimSpace(path)
	}
}

// GetExecutablePath returns the configured path, empty where none is set.
func (e *ExecutablePath) GetExecutablePath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.path
}

// StreamWork reports that the model started, starts the work and forwards its responses to the
// stream. A work that fails to start ends the stream with a stopped step that names the failure.
func StreamWork(
	ctx context.Context,
	stream *connect.ServerStream[AskAgentResponse],
	start func() (*Work, error),
) error {
	if err := stream.Send(&AskAgentResponse{
		Step: &AskAgentResponseModelStarted{ModelStarted: &ModelStarted{}},
	}); err != nil {
		return err
	}
	work, err := start()
	if err != nil {
		return stream.Send(&AskAgentResponse{Step: &AskAgentResponseStopped{Stopped: err.Error()}})
	}
	return ForwardResponses(ctx, work, stream)
}

// NewInstalledState answers GetAgentState: reachable where the program is installed, else
// unreachable with the message.
func NewInstalledState(isInstalled bool, message string) *connect.Response[GetAgentStateResponse] {
	if !isInstalled {
		return connect.NewResponse(&GetAgentStateResponse{Unreachable: message})
	}
	return connect.NewResponse(&GetAgentStateResponse{Unreachable: ""})
}
