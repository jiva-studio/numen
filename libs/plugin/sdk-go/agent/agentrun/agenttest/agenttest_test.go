package agenttest_test

import (
	"context"
	"os/exec"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

type runner struct {
	agentrun.Service
}

func (s *runner) Ask(ctx context.Context, req *agentrun.Request, _ func(*agentrun.Response) error) error {
	cmd := exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
	cmd.Dir = req.Root
	return cmd.Run()
}

func TestARecordingProgramRecordsTheFolderAndTheArguments(t *testing.T) {
	path, read := agenttest.RecordingProgram(t, "prog")
	root := t.TempDir()

	run := &runner{}
	agenttest.Ask(t, agenttest.Serve(t, run, run), &agentrun.Request{
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
