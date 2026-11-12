package codexcli

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
)

// Models asks Codex for the models it offers.
type Models struct{}

// List starts the program of command, or the Codex found on this machine when command is empty, in
// its app-server mode and returns the models it lists.
func (Models) List(ctx context.Context, command []string) ([]agentmodel.Model, error) {
	return askCodexModels(ctx, command)
}

// ReadStored returns the models in the file where Codex keeps the list it fetched last, or none.
func (Models) ReadStored() []agentmodel.Model {
	return readCodexCache()
}
