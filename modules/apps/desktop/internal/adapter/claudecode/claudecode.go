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
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	// vault's tools; the two the agent brings — searching the web and fetching
	// a page — are named where they are brought.
	Allowed []string
	// Words are how the tools this vault serves are spoken about, by the name
	// the agent calls them. A tool that is not here is named as it named
	// itself.
	Words map[string]ToolDeclaration
	// Drafting is how a change this agent is making is drawn before it lands.
	Drafting Drafting
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

	// carried is the session each conversation is on so far, under the name the
	// task gave its conversation. The next question of a conversation is asked
	// in the session carried for it, and a question that named no conversation
	// is asked in none.
	carried struct {
		sync.Mutex
		sessions map[string]string
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

// Finish ends a conversation: everything still being answered in it stops, and
// the session it was on is let go of once nothing is left to write one.
func (a *Agent) Finish(_ context.Context, conversation string) error {
	if conversation == "" {
		return nil
	}

	a.taken.Lock()
	answering := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		if w.conversation == conversation {
			answering = append(answering, w)
		}
	}
	a.taken.Unlock()

	var failed error
	for _, w := range answering {
		if err := w.Stop(); err != nil && failed == nil {
			failed = err
		}
	}

	a.carried.Lock()
	defer a.carried.Unlock()
	delete(a.carried.sessions, conversation)
	return failed
}

// Carrying is the session the next question of this conversation is asked in,
// empty for a conversation nothing has been asked in yet.
func (a *Agent) Carrying(conversation string) string {
	if conversation == "" {
		return ""
	}
	a.carried.Lock()
	defer a.carried.Unlock()
	return a.carried.sessions[conversation]
}

// carrying is what one run tells the session it is on to. It is kept under
// that run's conversation, and a run that named none is kept nowhere.
func (a *Agent) carrying(conversation string) func(string) {
	if conversation == "" {
		return func(string) {}
	}
	return func(session string) {
		a.carried.Lock()
		defer a.carried.Unlock()
		if a.carried.sessions == nil {
			a.carried.sessions = map[string]string{}
		}
		a.carried.sessions[conversation] = session
	}
}

// Name is what the server this agent is served by calls itself, and the prefix
// its tools arrive under.
const Name = "numen"

// DefaultTurns is how many times an agent may go round on one task.
const DefaultTurns = 30

// Take starts the agent on a task.
func (a *Agent) Take(ctx context.Context, task port.Task) (port.Work, error) {
	if a.Tools.URL == "" || a.Tools.Token == "" {
		return nil, errors.New("no tools to give an agent")
	}

	configuration, err := a.configuration()
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
	name, rest := a.command()
	cmd := exec.CommandContext(running, name, append(rest, a.arguments(task, configuration)...)...)
	cmd.Dir = a.Root
	cmd.Env = environment(os.Environ())
	detach(cmd)

	// The whole group goes, and the pipes are let go of shortly after: a
	// grandchild holding the child's error output keeps a wait from returning.
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = 2 * time.Second

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
	w := &work{
		cmd:          cmd,
		stop:         stop,
		conversation: task.Conversation,
		steps:        make(chan port.Step, 16),
	}
	if !a.hold(w) {
		stop()
		return nil, errors.New("this agent is closing")
	}
	if err := cmd.Start(); err != nil {
		a.letGo(w)
		stop()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	started = true
	w.reader.Add(1)
	go func() {
		defer w.reader.Done()
		defer a.letGo(w)
		defer close(w.steps)
		defer os.Remove(configuration)

		failed := read(running, out, w.steps, a.Words, a.carrying(task.Conversation), a.Drafting)

		err := cmd.Wait()
		if running.Err() != nil {
			// Let go of: the person closed the panel or asked something else.
			return
		}
		if failed == "" {
			failed = reason(err, said.String())
		}
		select {
		case w.steps <- port.Step{Kind: port.StepStopped, Detail: failed}:
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
		return installed(), nil
	}
	return a.Command[0], a.Command[1:]
}

// places are where the command line is looked for when the path does not name
// it.
//
// An application opened from a desktop is given the system path alone, so every
// folder an installer writes to is named here. A leading ~ is this person's
// home, and a * is expanded.
//
// A mac carries programs inside application bundles, and the bundle the
// command line's installer leaves there holds a link to it.
var places = []string{
	"~/.local/bin/claude",
	"~/.claude/local/claude",
	"~/Applications/Claude Code URL Handler.app/Contents/MacOS/claude",
	"/Applications/Claude Code URL Handler.app/Contents/MacOS/claude",
	"~/.bun/bin/claude",
	"~/.volta/bin/claude",
	"~/.npm-global/bin/claude",
	"~/.nvm/versions/node/*/bin/claude",
	"~/.nix-profile/bin/claude",
	"/opt/homebrew/bin/claude",
	"/usr/local/bin/claude",
	"/run/current-system/sw/bin/claude",
	"/nix/var/nix/profiles/default/bin/claude",
}

// installed is the command line to start: the path first, then the places.
//
// The bare name is the answer when it is nowhere, and starting that says it is
// not installed.
func installed() string {
	if named, err := exec.LookPath("claude"); err == nil {
		return named
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if found := found(home, places); found != "" {
		return found
	}
	return "claude"
}

// found is the first of places that is a program this machine can run. Empty
// says none of them is.
func found(home string, places []string) string {
	for _, place := range places {
		if strings.HasPrefix(place, "~/") {
			if home == "" {
				continue
			}
			place = filepath.Join(home, place[2:])
		}
		matches, err := filepath.Glob(place)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if runnable(match) {
				return match
			}
		}
	}
	return ""
}

// runnable is a file with an execute bit on it, held under the name it was
// looked for by.
//
// A mac filesystem answers to a name in any case, so the folder is asked which
// name it keeps.
func runnable(path string) bool {
	about, err := os.Stat(path)
	if err != nil || !about.Mode().IsRegular() || about.Mode().Perm()&0o111 == 0 {
		return false
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	name := filepath.Base(path)
	return slices.ContainsFunc(entries, func(e os.DirEntry) bool { return e.Name() == name })
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
const brought = "WebSearch"

// arguments are what the agent is started with.
//
// Only what the command line documents: the answer as one JSON object per line,
// this vault's tools and no other server's, and the tools named in Allowed
// approved ahead of the run. The question itself is not here — it goes on the
// input, where nothing reads it for options.
//
// The one tool it brings is the search; every other built-in is disabled. An
// agent works this vault through the tools this vault serves, and
// every one of those goes through a use case that says what a note is and keeps
// the index level with the file.
func (a *Agent) arguments(task port.Task, configuration string) []string {
	turns := a.Turns
	if turns <= 0 {
		turns = DefaultTurns
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--strict-mcp-config",
		"--mcp-config", configuration,
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
	if session := a.Carrying(task.Conversation); session != "" {
		args = append(args, "--resume", session)
	}
	return args
}

// dropped names the environment variables that describe a Claude Code session
// somebody else is running.
//
// A window is not one, so these are stripped from the environment the agent is
// started with. What says how to reach a model is not here: that belongs to the
// installation and is passed on.
var dropped = []string{
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
		if found && slices.Contains(dropped, name) {
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
func manners(task port.Task) string {
	var b strings.Builder
	b.WriteString("You are answering inside the application the person keeps these notes in, ")
	b.WriteString("beside the note they are looking at.\n\n")
	b.WriteString("Call a note by its title. Never show a path, a file name or an extension: ")
	b.WriteString("they are how the tools address a note and mean nothing to the person.\n")
	b.WriteString("Keep it short. They are reading in a narrow panel, not a terminal.\n")

	// The path is not written here. A note in a synced vault is named by
	// whoever synced it, and a name in the system prompt is read as
	// instruction. `window_tab_list` names it as a tool's answer, which is data.
	if task.Focus != "" {
		b.WriteString("\nA task that says \"this note\" means the one they are looking at, ")
		b.WriteString("which `window_tab_list` names.\n")
	}
	return b.String()
}

// servers is the one server this agent is given, written the way the command
// line reads it.
func (a *Agent) servers() ([]byte, error) {
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

// configuration writes the server this agent is given to a file of its own and
// answers with its path.
//
// The configuration carries the bearer token for this vault's tools. The file
// is this user's to read and nobody else's, and a command line is not, so the
// path is what the child is given. Whoever makes it removes it.
func (a *Agent) configuration() (string, error) {
	written, err := a.servers()
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

// prefix is what a tool of this vault's server is called under once it reaches
// an agent.
const prefix = "mcp__" + Name + "__"

// Tool is what a tool of this vault's server is called once it reaches an
// agent.
func Tool(name string) string { return prefix + name }

// ToolDeclaration is how one tool is spoken about to a person: what it calls
// itself and what a call of it does to the vault. Both are the tool's own
// declaration, read from what the server serves.
type ToolDeclaration struct {
	Title string
	// Kind is what a call of this tool does to the vault. A tool that declares
	// nothing about it is port.StepToolCall.
	Kind port.StepKind
	// Arguments is how a call of it is read while it is being written.
	Arguments Arguments
}

// Arguments are the names this tool's own arguments arrive under. Nothing here
// is shown to anybody: they are what a call half written is read for the value
// that is.
type Arguments struct {
	// About names the argument that says what a call was about.
	About string
	// Element names the field of one element that says which element it is, for
	// a call that takes a collection.
	Element string
	// Match and Text name the arguments carrying the text a call replaces
	// and what it puts in that text's place. Both are empty for a call that
	// replaces no stretch.
	Match string
	Text  string
}

// work is one task being worked, and what stops it.
type work struct {
	cmd  *exec.Cmd
	stop func()
	// conversation is the thread of talk this task was asked in, empty for a
	// question asked in none. Finishing that conversation stops this work.
	conversation string
	steps        chan port.Step
	reader       sync.WaitGroup
	once         sync.Once
}

func (w *work) Steps() <-chan port.Step { return w.steps }

// Stop ends the child and everything it started, then waits for the reading to
// finish so that nothing of this work is still running when it returns.
//
// Cancelling is the one way the child is ended, here and wherever the context
// this work was taken with is cancelled.
func (w *work) Stop() error {
	w.once.Do(w.stop)
	w.reader.Wait()
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

// Drafting is how a change is drawn while the call making it is still being
// written. Nothing here is a write: the vault is what changes a note, and this
// only says what is on its way.
type Drafting struct {
	// Report is told each time more of the change has arrived.
	Report func(ctx context.Context, said domain.Edit)
	// Location says where a stretch stands in a note, and whether it stands in
	// exactly one place. A stretch that stands nowhere or twice is not drawn.
	Location func(ctx context.Context, path, stood string) (from, to int, unique bool)
	// Now is the clock the pace is kept by.
	Now func() time.Time
}

// drawing reports whether anything can be drawn at all.
func (d Drafting) drawing() bool { return d.Report != nil && d.Location != nil }

func (d Drafting) now() time.Time {
	if d.Now == nil {
		return time.Now()
	}
	return d.Now()
}
