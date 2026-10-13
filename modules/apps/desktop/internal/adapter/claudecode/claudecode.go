// Package claudecode reaches the agent a person already has installed.
//
// The command line is started as a child process, given this vault's tools over
// the port the application is already serving them on, and read back line by
// line. Nothing about the vault is decided here: the tools do that, and this
// only carries a task to them and the answer back.
package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/runner"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Endpoint is where the agent reaches this vault's tools, and what it must
// present to be let in.
type Endpoint struct {
	URL   string
	Token string
}

// Options is what Claude Code is made from.
type Options struct {
	// Settings is how the installation runs it.
	Settings agent.Claude
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault.
	Tools Endpoint
	// ToolDeclarations are how the tools this vault serves are spoken about, by the name the agent
	// calls them.
	ToolDeclarations map[string]ToolDeclaration
	// Preview is how a change this agent is making is drawn before it lands.
	Preview EditPreview
	// ErrorHandler is told what the agent wrote to its error output when something went wrong.
	ErrorHandler func(error)
}

// New makes Claude Code, waiting to be asked something.
func New(opts Options) *Agent {
	return &Agent{
		Command:                  opts.Settings.Command,
		Root:                     opts.Root,
		Tools:                    opts.Tools,
		ToolDeclarations:         opts.ToolDeclarations,
		Preview:                  opts.Preview,
		Model:                    opts.Settings.Model,
		ShouldReadHooksAndSkills: opts.Settings.ShouldReadHooksAndSkills,
		ErrorHandler:             opts.ErrorHandler,
	}
}

// Agent is the command line, waiting to be asked something.
type Agent struct {
	// Command starts it. Empty means `claude` from the path.
	Command []string
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault.
	Tools Endpoint
	// ToolDeclarations are how the tools this vault serves are spoken about, by the name
	// the agent calls them. A tool that is not here is named as it named
	// itself.
	ToolDeclarations map[string]ToolDeclaration
	// Preview is how a change this agent is making is drawn before it lands.
	Preview EditPreview
	// Model is which model answers, by the name the command line knows it as.
	// Empty leaves the choice to the installation the agent belongs to.
	Model string
	// ShouldReadHooksAndSkills lets the agent read what this machine holds for it:
	// hooks, skills, standing instructions, plugins. What a vault carries is
	// refused whether this is set or not.
	ShouldReadHooksAndSkills bool
	// ErrorHandler is told what the agent wrote to its error output when
	// something went wrong.
	ErrorHandler func(error)

	// registry holds the work this agent started and the session each conversation is on. The
	// child is in a process group of its own, so nothing else ends it.
	registry runner.Registry
}

// Close stops every agent this one started and waits for them.
func (a *Agent) Close() error { return a.registry.Close() }

// EndConversation stops everything still being answered in the conversation and lets go of its
// session.
func (a *Agent) EndConversation(ctx context.Context, conversation string) error {
	return a.registry.EndConversation(ctx, conversation)
}

// DefaultMaxTurns is how many times an agent may go round on one task.
const DefaultMaxTurns = 100

// StartRun starts the agent on a task.
func (a *Agent) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	if a.Tools.URL == "" || a.Tools.Token == "" {
		return nil, errors.New("no tools to give an agent")
	}

	configuration, err := a.writeConfigFile()
	if err != nil {
		return nil, fmt.Errorf("write the tools an agent is given: %w", err)
	}
	// The child reads the file as it starts and the run holds it until the
	// process is done with it.
	started := false
	defer func() {
		if !started {
			os.Remove(configuration)
		}
	}()

	running, stop := context.WithCancel(ctx)
	name, rest := a.getCommand()
	cmd := exec.CommandContext(running, name, append(rest, a.buildArguments(task, configuration)...)...)
	cmd.Dir = a.Root
	cmd.Env = buildEnvironment(os.Environ())
	runner.Supervise(cmd)

	// The question goes on the input. A question is a person's own words and a
	// note's, and words on a command line are read for options first.
	cmd.Stdin = strings.NewReader(task.Question)

	out, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, err
	}
	var said strings.Builder
	cmd.Stderr = &said

	// Held before it is started, so a close cannot pass between the two and
	// leave a child nothing reaches.
	w := runner.NewWork(task.Conversation, 16, stop)
	if !a.registry.Add(w) {
		stop()
		return nil, errors.New("this agent is closing")
	}
	if err := cmd.Start(); err != nil {
		a.registry.Remove(w)
		stop()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	started = true
	w.Begin()
	go a.follow(running, w, child{cmd: cmd, name: name, out: out, said: &said, configuration: configuration}, task.Conversation)
	return w, nil
}

// child is the process an agent runs as: what it writes, and the file it was
// given its tools in.
type child struct {
	cmd           *exec.Cmd
	name          string
	out           io.Reader
	said          *strings.Builder
	configuration string
}

// follow reads the steps of the process until its output ends, and says how it
// stopped unless the run was let go of.
func (a *Agent) follow(running context.Context, w *runner.Work, c child, conversation string) {
	defer w.End()
	defer a.registry.Remove(w)
	defer close(w.Output)
	defer os.Remove(c.configuration)

	failed := readStream(running, c.out, w.Output, a.ToolDeclarations, a.makeSessionRecorder(conversation), a.Preview)

	err := c.cmd.Wait()
	if running.Err() != nil {
		// Let go of: the person closed the panel or asked something else.
		return
	}
	if failed == "" {
		failed = getFailureMessage(err, c.said.String())
	}
	select {
	case w.Output <- port.Step{Kind: domain.StepStop, Detail: failed}:
	case <-running.Done():
	}
	if err != nil && a.ErrorHandler != nil {
		a.ErrorHandler(fmt.Errorf("%s: %w: %s", c.name, err, strings.TrimSpace(c.said.String())))
	}
}

func (a *Agent) getCommand() (string, []string) {
	if len(a.Command) == 0 {
		return agent.FindClaude(), nil
	}
	return a.Command[0], a.Command[1:]
}

// brought is the tools the agent may use besides this vault's own: it may look
// something up, and it may not touch this machine. Every other built-in — a
// shell, a file writer, a file reader — is absent.
//
// Looking something up is a search and not a fetch. A note may have been
// written by anybody and the agent reads notes, so a tool that goes to an
// address the text names is an address the text chooses: the vault leaves in
// the request. A search names no address, and the words of it reach the model
// that is reading them already.
const externalTool = "WebSearch"

// buildArguments returns what the command line is started with.
func (a *Agent) buildArguments(task port.Task, configuration string) []string {
	turns := task.MaxSteps
	if turns <= 0 {
		turns = DefaultMaxTurns
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--strict-mcp-config",
		"--mcp-config", configuration,
		"--tools", externalTool,
		"--permission-mode", "dontAsk",
		"--max-turns", fmt.Sprint(turns),
	}
	if task.Instructions != "" {
		args = append(args, "--append-system-prompt", task.Instructions)
	}
	// A hook is a shell command, and a settings file inside a vault is a vault
	// telling this machine what to run. Naming the sources read is what refuses
	// them: none by default, and only the person's own when they ask. A vault's
	// are refused either way.
	// The sources are named one by one because this vault's own tools arrive on
	// a command line and are read as a customisation like any other.
	sources := ""
	if a.ShouldReadHooksAndSkills {
		sources = "user"
	}
	args = append(args, "--setting-sources", sources)
	model := a.Model
	if task.Model != "" {
		model = task.Model
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	// The search is allowed alongside this vault's tools. Naming a tool on
	// --tools offers it; the allowance is what lets it be called, and nothing
	// outside the allowance is called at all under this mode.
	args = append(args, "--allowedTools", strings.Join(getAllowedTools(task), ","))
	if session := a.registry.GetSession(task.Conversation); session != "" {
		args = append(args, "--resume", session)
	}
	return args
}

// getServersConfig returns the one server this agent is given, written the way the command
// line reads it.
func (a *Agent) getServersConfig() ([]byte, error) {
	type server struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	config := struct {
		Servers map[string]server `json:"mcpServers"`
	}{Servers: map[string]server{Name: {
		Type:    "http",
		URL:     a.Tools.URL,
		Headers: map[string]string{"Authorization": "Bearer " + a.Tools.Token},
	}}}

	return json.Marshal(config)
}

// writeConfigFile writes the server this agent is given to a file of its own and
// answers with its path.
//
// The configuration carries the bearer token for this vault's tools. The file
// is this user's to read and nobody else's, and a command line is not, so the
// path is what the child is given. Whoever makes it removes it.
func (a *Agent) writeConfigFile() (string, error) {
	written, err := a.getServersConfig()
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "numen-tools-*.json")
	if err != nil {
		return "", err
	}
	at := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(at)
		return "", err
	}
	if _, err := file.Write(written); err != nil {
		file.Close()
		os.Remove(at)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(at)
		return "", err
	}
	return at, nil
}

// getFailureMessage is what to tell the person when the agent stopped.
func getFailureMessage(err error, said string) string {
	if err == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(said); trimmed != "" {
		return getLastLine(trimmed)
	}
	return err.Error()
}

// getLastLine is the part of the error output worth showing: what a command line
// says last is what went wrong.
func getLastLine(said string) string {
	lines := strings.Split(said, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return said
}

// EditPreview is how a change is drawn while the call making it is still being
// written. Nothing here is a write: the vault is what changes a note, and this
// only says what is on its way.
type EditPreview struct {
	// Emit is told each time more of the change has arrived.
	Emit func(ctx context.Context, said domain.Edit)
	// Locate says where a stretch stands in a note, and whether it stands in
	// exactly one place. A stretch that stands nowhere or twice is not drawn.
	Locate func(ctx context.Context, path, stood string) (from, to int, unique bool)
	// Now is the clock the pace is kept by.
	Now func() time.Time
}

// isEnabled reports whether anything can be drawn at all.
func (d EditPreview) isEnabled() bool { return d.Emit != nil && d.Locate != nil }

func (d EditPreview) getNow() time.Time {
	if d.Now == nil {
		return time.Now()
	}
	return d.Now()
}

// makeSessionRecorder is what one run tells the session it is on to. It is kept
// under that run's conversation, and a run that named none is kept nowhere.
func (a *Agent) makeSessionRecorder(conversation string) func(string) {
	return func(session string) { a.registry.SetSession(conversation, session) }
}

// getAllowedTools returns the tools the agent may call: the search it brings, and the vault's
// tools the task names. A tool absent from it is refused under the mode the agent runs in.
func getAllowedTools(task port.Task) []string {
	allowed := make([]string, 0, len(task.Tools))
	for _, name := range task.Tools {
		allowed = append(allowed, ToolName(name))
	}
	slices.Sort(allowed)
	return append([]string{externalTool}, allowed...)
}
