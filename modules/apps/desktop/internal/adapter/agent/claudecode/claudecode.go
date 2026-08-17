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
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"
)

// Endpoint is where the agent reaches this vault's tools, and what it must
// present to be let in.
type Endpoint struct {
	URL   string
	Token string
}

// Agent is the command line, waiting to be asked something.
type Agent struct {
	// Command starts it. Empty means `claude` from the path.
	Command []string
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault.
	Tools Endpoint
	// Allowed are the tools it may use without being asked. It names this
	// vault's tools: the agent is started with none of its own, so there is
	// nothing else to approve.
	Allowed []string
	// Words are how the tools this vault serves are spoken about, by the name
	// the agent calls them. A tool that is not here is named as it named
	// itself.
	Words map[string]Words
	// Model is which model answers, by the name the command line knows it as.
	// Empty leaves the choice to the installation the agent belongs to.
	Model string
	// ReadsHooksAndSkills lets the agent read what this machine holds for it:
	// hooks, skills, standing instructions, plugins. What a vault carries is
	// refused whether this is set or not.
	ReadsHooksAndSkills bool
	// Turns is how many times the agent may go to the model before it is stopped.
	Turns int
	// Trouble is told what the agent wrote to its error output when something
	// went wrong.
	Trouble func(error)

	// carried is the conversation so far. The next question is asked in it.
	carried struct {
		sync.Mutex
		session string
	}

	// taken is the work this agent started and has not been told is over. The
	// child is in a process group of its own, so nothing else ends it.
	taken struct {
		sync.Mutex
		running map[*work]bool
		shut    bool
	}
}

// hold keeps the work so that closing can reach it.
func (a *Agent) hold(w *work) bool {
	a.taken.Lock()
	defer a.taken.Unlock()

	if a.taken.shut {
		return false
	}
	if a.taken.running == nil {
		a.taken.running = map[*work]bool{}
	}
	a.taken.running[w] = true
	return true
}

func (a *Agent) letGo(w *work) {
	a.taken.Lock()
	defer a.taken.Unlock()
	delete(a.taken.running, w)
}

// Close stops every agent this one started and waits for them.
//
// A task outlives the window otherwise: the child is started in its own process
// group, so it is left running by whatever ends this process, and it goes on
// writing to the vault with nobody watching.
func (a *Agent) Close() error {
	a.taken.Lock()
	a.taken.shut = true
	running := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		running = append(running, w)
	}
	a.taken.running = nil
	a.taken.Unlock()

	var failed error
	for _, w := range running {
		if err := w.Stop(); err != nil && failed == nil {
			failed = err
		}
	}
	return failed
}

// Carrying is the conversation the next question is asked in.
func (a *Agent) Carrying() string {
	a.carried.Lock()
	defer a.carried.Unlock()
	return a.carried.session
}

func (a *Agent) carry(session string) {
	a.carried.Lock()
	defer a.carried.Unlock()
	a.carried.session = session
}

// Name is what the server this agent is served by calls itself, and the prefix
// its tools arrive under.
const Name = "numen"

// DefaultTurns is how many times an agent may go round on one task.
const DefaultTurns = 30

// Take starts the agent on a task.
func (a *Agent) Take(ctx context.Context, task agent.Task) (agent.Work, error) {
	if a.Tools.URL == "" || a.Tools.Token == "" {
		return nil, errors.New("no tools to give an agent")
	}

	running, stop := context.WithCancel(ctx)
	name, rest := a.command()
	cmd := exec.CommandContext(running, name, append(rest, a.arguments(task)...)...)
	cmd.Dir = a.Root
	cmd.Env = environment(os.Environ())
	detach(cmd)

	// The whole group goes, and the pipes are let go of shortly after: a
	// grandchild holding the child's error output keeps a wait from returning.
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = 2 * time.Second

	out, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, err
	}
	var said strings.Builder
	cmd.Stderr = &said

	// Held before it is started, so a close cannot pass between the two and
	// leave a child nothing reaches.
	w := &work{cmd: cmd, stop: stop, steps: make(chan agent.Step, 16)}
	if !a.hold(w) {
		stop()
		return nil, errors.New("this agent is closing")
	}
	if err := cmd.Start(); err != nil {
		a.letGo(w)
		stop()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	w.reading.Add(1)
	go func() {
		defer w.reading.Done()
		defer a.letGo(w)
		defer close(w.steps)

		failed := read(running, out, w.steps, a.Words, a.carry)

		err := cmd.Wait()
		if running.Err() != nil {
			// Let go of: the person closed the panel or asked something else.
			return
		}
		if failed == "" {
			failed = reason(err, said.String())
		}
		select {
		case w.steps <- agent.Step{Kind: agent.Stopped, Failed: failed}:
		case <-running.Done():
		}
		if err != nil && a.Trouble != nil {
			a.Trouble(fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(said.String())))
		}
	}()
	return w, nil
}

func (a *Agent) command() (string, []string) {
	if len(a.Command) == 0 {
		return "claude", nil
	}
	return a.Command[0], a.Command[1:]
}

// brought is the tools the agent may use besides this vault's own: it may look
// something up, and it may not touch this machine. Every other built-in — a
// shell, a file writer, a file reader — is absent.
const brought = "WebSearch,WebFetch"

// arguments are what the agent is started with.
//
// Only what the command line documents: the task on the command line, the
// answer as one JSON object per line, this vault's tools and no other server's,
// and the tools named in Allowed approved ahead of the run.
//
// The tools it brings are the two that reach the web; every other built-in is
// disabled. An agent works this vault through the tools this vault serves, and
// every one of those goes through a use case that says what a note is and keeps
// the index level with the file.
func (a *Agent) arguments(task agent.Task) []string {
	turns := a.Turns
	if turns <= 0 {
		turns = DefaultTurns
	}

	args := []string{
		"-p", task.Asked,
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--strict-mcp-config",
		"--mcp-config", a.servers(),
		"--tools", brought,
		"--permission-mode", "dontAsk",
		"--max-turns", fmt.Sprint(turns),
		"--append-system-prompt", manners(task),
	}
	// A hook is a shell command, and a settings file inside a vault is a vault
	// telling this machine what to run. Naming the sources read is what refuses
	// them: none by default, and only the person's own when they ask. A vault's
	// are refused either way.
	//
	// Named rather than turned off wholesale, because turning every
	// customisation off takes this vault's own tools with it — they arrive on a
	// command line and are read as a customisation like any other.
	sources := ""
	if a.ReadsHooksAndSkills {
		sources = "user"
	}
	args = append(args, "--setting-sources", sources)
	if a.Model != "" {
		args = append(args, "--model", a.Model)
	}
	if len(a.Allowed) > 0 {
		args = append(args, "--allowedTools", strings.Join(a.Allowed, ","))
	}
	if session := a.Carrying(); session != "" {
		args = append(args, "--resume", session)
	}
	return args
}

// session names the variables that describe a Claude Code session somebody else
// is running.
//
// A window is not one, so these are stripped from the environment the agent is
// started with. What says how to reach a model is not here: that belongs to the
// installation and is passed on.
var session = []string{
	"CLAUDECODE",
	"CLAUDE_CODE_SESSION_ID",
	"CLAUDE_CODE_CHILD_SESSION",
	"CLAUDE_CODE_ENTRYPOINT",
	"CLAUDE_CODE_EXECPATH",
	"CLAUDE_CODE_MESSAGING_SOCKET",
	"CLAUDE_CODE_MESSAGING_TOKEN",
	"CLAUDE_PID",
	"CLAUDE_EFFORT",
}

// environment is what the agent is started with: everything the machine holds,
// less what belongs to a session it is not part of.
func environment(held []string) []string {
	out := make([]string, 0, len(held))
	for _, entry := range held {
		name, _, found := strings.Cut(entry, "=")
		if found && slices.Contains(session, name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// manners is what the agent is told about the person it is answering.
//
// A path is how a tool names a note. The person named it by writing a title on
// it, and that is the name they know it by: a note is "Harmonic oscillator",
// never `physics/classical/Harmonic oscillator.md`.
func manners(task agent.Task) string {
	var b strings.Builder
	b.WriteString("You are answering inside the application the person keeps these notes in, ")
	b.WriteString("beside the note they are looking at.\n\n")
	b.WriteString("Call a note by its title. Never show a path, a file name or an extension: ")
	b.WriteString("they are how the tools address a note and mean nothing to the person.\n")
	b.WriteString("Keep it short. They are reading in a narrow panel, not a terminal.\n")

	if task.Focus != "" {
		fmt.Fprintf(&b, "\nThe note in front of them is at %s. A task that says \"this note\" means that one.\n", task.Focus)
	}
	return b.String()
}

// servers is the one server this agent is given, written the way the command
// line reads it.
func (a *Agent) servers() string {
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

	written, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(written)
}

// prefix is what a tool of this vault's server is called under once it reaches
// an agent.
const prefix = "mcp__" + Name + "__"

// Tool is what a tool of this vault's server is called once it reaches an
// agent.
func Tool(name string) string { return prefix + name }

// Words are how one tool is spoken about to a person: what it calls itself,
// and which of its arguments says what a call was about. Both are the tool's
// own declaration, read from what the server serves.
type Words struct {
	Title string
	About string
	// Inside names the field of one element that says which element it is, for
	// a call that takes a collection.
	Inside string
}

// work is one task being worked, and what stops it.
type work struct {
	cmd     *exec.Cmd
	stop    func()
	steps   chan agent.Step
	reading sync.WaitGroup
	once    sync.Once
}

func (w *work) Steps() <-chan agent.Step { return w.steps }

// Stop ends the child and everything it started, then waits for the reading to
// finish so that nothing of this work is still running when it returns.
//
// Cancelling is the one way the child is ended, here and wherever the context
// this work was taken with is cancelled.
func (w *work) Stop() error {
	w.once.Do(w.stop)
	w.reading.Wait()
	return nil
}

// reason is what to tell the person when the agent stopped.
func reason(err error, said string) string {
	if err == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(said); trimmed != "" {
		return lastLine(trimmed)
	}
	return err.Error()
}

// lastLine is the part of the error output worth showing: what a command line
// says last is what went wrong.
func lastLine(said string) string {
	lines := strings.Split(said, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return said
}
