package agenttest_test

import (
	"context"
	"os/exec"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type runner struct {
	agentrun.Service
}

func (s *runner) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	cmd := exec.CommandContext(ctx, req.Msg.GetCommand()[0], req.Msg.GetCommand()[1:]...)
	cmd.Dir = req.Msg.GetRoot()
	return cmd.Run()
}

func TestARecordingProgramRecordsTheFolderAndTheArguments(t *testing.T) {
	path, read := agenttest.RecordingProgram(t, "prog")
	root := t.TempDir()

	agenttest.Ask(t, agenttest.Serve(t, &runner{}), &numenv1.AskAgentRequest{
		Command: []string{path, "--flag", "two words"},
		Root:    root,
	})

	dir, argv := read()
	if !agenttest.IsDir(dir, root) {
		t.Errorf("ran in %q, want %q", dir, root)
	}
	if want := []string{"--flag", "two words"}; !slices.Equal(argv, want) {
		t.Errorf("the arguments are %v, want %v", argv, want)
	}
}

func TestQuoteShellEscapesSingleQuotes(t *testing.T) {
	if got := agenttest.QuoteShell("it's"); got != `'it'\''s'` {
		t.Errorf("got %s", got)
	}
}
