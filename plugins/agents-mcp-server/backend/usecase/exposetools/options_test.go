package exposetools

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/instructions"
)

func TestServerOptionsCarryTheServerLine(t *testing.T) {
	t.Parallel()

	srvOpts := newServerOptions()
	if srvOpts == nil || srvOpts.Instructions != instructions.Server {
		t.Fatalf("unexpected server options: %+v", srvOpts)
	}
}
