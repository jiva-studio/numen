package claudecode_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent/claudecode"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// started is an agent whose command line is a script printing what it was told
// to print. What is tested is the reading and the stopping: the tools are the
// server's business and the answering is the model's.
func started(t *testing.T, prints string) port.Work {
	t.Helper()

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\ncat <<'SAID'\n" + prints + "\nSAID\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
		Words: map[string]claudecode.Words{
			claudecode.Tool("note_search"): {Title: "Search notes", About: "query"},
			claudecode.Tool("note_create"): {Title: "Create a note", About: "notes", Inside: "title"},
			claudecode.Tool("note_write"):  {Title: "Write a note", About: "path", Kind: port.StepEdit},
		},
	}
	work, err := claude.Take(t.Context(), port.Task{Asked: "what is here?"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.Stop() })
	return work
}

// heard is every step of a piece of work, in order.
func heard(t *testing.T, work port.Work) []port.Step {
	t.Helper()

	var steps []port.Step
	for step := range work.Steps() {
		steps = append(steps, step)
	}
	return steps
}

const connected = `{"type":"system","subtype":"init","session_id":"s1",` +
	`"mcp_servers":[{"name":"numen","status":"connected"}]}`

func TestSaysWhatTheAgentSaid(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Two notes."}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false,"result":"Two notes."}`)

	steps := heard(t, work)
	if len(steps) != 2 {
		t.Fatalf("expected saying and stopping, got %d steps: %+v", len(steps), steps)
	}
	if steps[0].Kind != port.StepSaying || steps[0].Text != "Two notes." {
		t.Errorf("first step is %+v", steps[0])
	}
	if steps[1].Kind != port.StepStopped || steps[1].Failed != "" {
		t.Errorf("last step is %+v", steps[1])
	}
}

func TestNamesAToolAsItNamedItself(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__numen__note_search",`+
		`"input":{"query":"entropy"}}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != port.StepCalling || steps[0].Tool != "Search notes" || steps[0].About != "entropy" {
		t.Errorf("expected the tool's own title and what it was asked, got %+v", steps[0])
	}
}

// A tool this vault does not serve is named as the agent named it: there is
// nothing declared here to read it by.
func TestNamesAToolItWasNotToldAbout(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Tool != "Bash" || steps[0].About != "" {
		t.Errorf("steps are %+v", steps)
	}
}

// Words arrive a piece at a time, and a call is shown once it is whole.
func TestReadsWordsAndCallsAsTheyAreWritten(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Two "}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"notes."}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"mcp__numen__note_search"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"query\":\"ent"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"ropy\"}"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Two notes."}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)

	var said []string
	var calls []port.Step
	for _, step := range steps {
		switch step.Kind {
		case port.StepSaying:
			said = append(said, step.Text)
		case port.StepCalling:
			calls = append(calls, step)
		}
	}

	// The pieces, and not the whole that follows every piece of it.
	if !slices.Equal(said, []string{"Two ", "notes."}) {
		t.Errorf("the words said are %q", said)
	}
	// A call is reported as it is written and again once it is whole. What it is
	// about is known by then.
	if len(calls) == 0 {
		t.Fatal("no call")
	}
	if last := calls[len(calls)-1]; last.About != "entropy" {
		t.Errorf("call is %+v", last)
	}
}

func TestPassesOverALineItCannotRead(t *testing.T) {
	work := started(t, connected+"\n"+
		"not json at all\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"still here"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != port.StepSaying || steps[0].Text != "still here" {
		t.Errorf("a line it could not read stopped the work: %+v", steps)
	}
}

func TestSaysWhyItStopped(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"result","subtype":"error_max_turns","is_error":true,"result":"went round too many times"}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != port.StepStopped || last.Failed != "went round too many times" {
		t.Errorf("last step is %+v", last)
	}
}

func TestSaysWhenTheVaultDidNotReachTheAgent(t *testing.T) {
	work := started(t, `{"type":"system","subtype":"init","session_id":"s1","mcp_servers":[]}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != port.StepStopped || !strings.Contains(last.Failed, "without this vault") {
		t.Errorf("an agent that never got the tools answered anyway: %+v", last)
	}
}

// A message of somebody else's shape must not stop the reading: what a user
// message carries is not what an assistant message carries.
func TestReadsPastAMessageOfAnotherShape(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"user","message":{"role":"user","content":"a string, not blocks"}}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"after"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != port.StepSaying || steps[0].Text != "after" {
		t.Errorf("steps are %+v", steps)
	}
}

func TestStoppingLeavesNothingRunning(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"a word"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	if err := work.Stop(); err != nil {
		t.Fatal(err)
	}
	// Stopping twice is what a panel closed twice does.
	if err := work.Stop(); err != nil {
		t.Fatal(err)
	}
	for range work.Steps() {
		// Draining what was already read is fine; the channel must close.
	}
}

// A version that says nothing about servers is not a version that says the
// vault never arrived.
func TestSaysNothingWhenTheLineSaysNothingAboutServers(t *testing.T) {
	work := started(t, `{"type":"system","subtype":"init","session_id":"s1"}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"here"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != port.StepStopped || last.Failed != "" {
		t.Errorf("last step is %+v", last)
	}
}

// recorded is what the agent's command line was called with, from a script that
// writes down its arguments and answers nothing.
func recorded(t *testing.T) []string {
	t.Helper()
	return recordedWith(t, func(*claudecode.Agent) {})
}

// recordedWith is recorded with the agent changed before it is started.
func recordedWith(t *testing.T, change func(*claudecode.Agent)) []string {
	t.Helper()

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	written := filepath.Join(dir, "argv")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + written + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
		Allowed: []string{claudecode.Tool("*")},
	}
	change(&claude)
	work, err := claude.Take(t.Context(), port.Task{Asked: "what is here?"})
	if err != nil {
		t.Fatal(err)
	}
	// Drained so that the script has run and written before it is read.
	heard(t, work)

	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

// The agent may look something up and may not touch this machine, so the run
// names the whole set of tools it is started with.
func TestTheAgentBringsOnlyTheToolsItIsNamed(t *testing.T) {
	argv := recorded(t)

	at := -1
	for i, arg := range argv {
		if arg == "--tools" {
			at = i
		}
	}
	if at < 0 {
		t.Fatalf("nothing names the built-in tools: %q", argv)
	}
	if at+1 >= len(argv) {
		t.Fatal("--tools was given nothing")
	}

	named := strings.Split(argv[at+1], ",")
	// Named one at a time: a check that only counts passes when the set changes
	// to another set of the same size.
	if !slices.Equal(named, []string{"WebSearch", "WebFetch"}) {
		t.Errorf("the tools it brings are %q", named)
	}
	// Nothing that reads or writes this machine, whatever else is added.
	for _, refused := range []string{"Bash", "Write", "Edit", "Read", "Task", "NotebookEdit"} {
		if slices.Contains(named, refused) {
			t.Errorf("%s is a tool it brought", refused)
		}
	}
}

// One server, named in full, and no chance of another being read from the
// machine's own configuration.
func TestTheAgentReachesThisVaultAndNothingElse(t *testing.T) {
	argv := recorded(t)

	if !slices.Contains(argv, "--strict-mcp-config") {
		t.Errorf("another server's configuration may still be read: %q", argv)
	}
	config := ""
	for i, arg := range argv {
		if arg == "--mcp-config" && i+1 < len(argv) {
			config = argv[i+1]
		}
	}
	if !strings.Contains(config, "127.0.0.1:7717") {
		t.Errorf("the vault is not the server it was given: %q", config)
	}
}

// kinds is what a run said, as the kinds of its steps in order.
func kinds(steps []port.Step) []port.StepKind {
	out := make([]port.StepKind, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Kind)
	}
	return out
}

const (
	requesting = `{"type":"system","subtype":"status","status":"requesting"}`
	answered   = `{"type":"user","message":{"content":[{"type":"tool_result","content":"done"}]}}`
)

// The moment a request to the model begins is written into the stream, and it is
// the moment a wait starts.
func TestSaysWhenTheModelWasAskedSomething(t *testing.T) {
	steps := heard(t, started(t, strings.Join([]string{connected, requesting}, "\n")))

	if !slices.Contains(kinds(steps), port.StepThinking) {
		t.Errorf("nothing says the model was asked: %+v", steps)
	}
}

// A tool answering is the end of that tool, and the stream says so.
func TestSaysWhenTheToolAnswered(t *testing.T) {
	steps := heard(t, started(t, strings.Join([]string{connected, answered}, "\n")))

	if !slices.Contains(kinds(steps), port.StepAnswered) {
		t.Errorf("nothing says the tool finished: %+v", steps)
	}
}

// A call carrying the body of a note is written for minutes. It is named as it
// is reached for, and reported again as it is written.
func TestReportsACallWhileItIsStillBeingWritten(t *testing.T) {
	// Long enough to be reported more than once as it arrives.
	body := strings.Repeat("Игра в кости есть корень несчастья. ", 30)
	lines := []string{
		connected,
		`{"type":"stream_event","event":{"type":"content_block_start",` +
			`"content_block":{"type":"tool_use","name":"` + claudecode.Tool("note_create") + `"}}}`,
		delta(`{"notes":[{"title":"Vidura's warning","body":"`),
		delta(body),
		delta(body),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}
	steps := heard(t, started(t, strings.Join(lines, "\n")))

	var calls []port.Step
	for _, s := range steps {
		if s.Kind == port.StepCalling {
			calls = append(calls, s)
		}
	}
	if len(calls) < 3 {
		t.Fatalf("a call written over minutes was reported %d times: %+v", len(calls), calls)
	}
	if calls[0].Tool != "Create a note" {
		t.Errorf("first says %q", calls[0].Tool)
	}
	// The name is read out of arguments that have not finished arriving.
	if calls[1].About != "Vidura's warning" {
		t.Errorf("what it is writing is %q", calls[1].About)
	}
	if calls[len(calls)-1].Written <= calls[1].Written {
		t.Errorf("what has been written did not grow: %d then %d",
			calls[1].Written, calls[len(calls)-1].Written)
	}
}

// delta is one piece of a call's arguments as the stream writes it.
func delta(partial string) string {
	quoted, err := json.Marshal(partial)
	if err != nil {
		panic(err)
	}
	return `{"type":"stream_event","event":{"type":"content_block_delta",` +
		`"delta":{"type":"input_json_delta","partial_json":` + string(quoted) + `}}}`
}

// wrote is one message carrying a call that writes a note and a call that looks
// for one.
var wrote = `{"type":"assistant","message":{"content":[` +
	`{"type":"tool_use","id":"toolu_7","name":"` + claudecode.Tool("note_write") + `",` +
	`"input":{"path":"physics/entropy.md","body":"Two words."}},` +
	`{"type":"tool_use","id":"toolu_8","name":"` + claudecode.Tool("note_search") + `",` +
	`"input":{"query":"entropy"}}]}}`

// One call is reported as it is reached for and again as it is written, and what
// says those reports are one call is the name the agent gave it.
func TestEveryReportOfOneCallCarriesTheNameTheAgentGaveIt(t *testing.T) {
	// Long enough to be reported while it is still being written.
	body := strings.Repeat("Игра в кости есть корень несчастья. ", 20)
	lines := []string{
		connected,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":` +
			`{"type":"tool_use","id":"toolu_7","name":"` + claudecode.Tool("note_write") + `"}}}`,
		delta(`{"path":"physics/entropy.md","body":"` + body),
		delta(`"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}
	steps := heard(t, started(t, strings.Join(lines, "\n")))

	reports := 0
	for _, step := range steps {
		if step.Tool != "Write a note" {
			continue
		}
		reports++
		if step.Call != "toolu_7" {
			t.Errorf("a report of the call says %q", step.Call)
		}
	}
	if reports < 2 {
		t.Fatalf("one call was reported %d times: %+v", reports, steps)
	}
}

// What a call does to the vault is what a person watching it wants to know, and
// the tool's own declaration is what says so.
func TestSaysWhatACallDoesToTheVault(t *testing.T) {
	steps := heard(t, started(t, connected+"\n"+wrote))

	if steps[0].Kind != port.StepEdit {
		t.Errorf("a call that writes a note is %+v", steps[0])
	}
	// A tool that declared nothing about what it does is a call and no more.
	if steps[1].Kind != port.StepCalling {
		t.Errorf("a call that says nothing about itself is %+v", steps[1])
	}
}

// A client follows the agent by opening what it is working in, and a path is the
// only thing that says which note that is.
func TestSaysWhichNoteACallIsWorkingIn(t *testing.T) {
	steps := heard(t, started(t, connected+"\n"+wrote))

	if steps[0].Place.Path != "physics/entropy.md" {
		t.Errorf("the call is working in %+v", steps[0].Place)
	}
	// A query names no note, and nothing is opened for it.
	if at := steps[1].Place; at.Path != "" || at.Start != 0 || at.Length != 0 {
		t.Errorf("a search is working in %+v", at)
	}
}

// A hook is a shell command the agent's own program runs, and it is not a tool:
// nothing about the tools it may use has any bearing on it. A question typed
// into a panel is not asking for one.
func TestTheAgentReadsNothingThisMachineHoldsForIt(t *testing.T) {
	argv := recorded(t)

	at := slices.Index(argv, "--setting-sources")
	if at < 0 {
		t.Fatalf("every source is read, hooks and all: %q", argv)
	}
	if got := argv[at+1]; got != "" {
		t.Errorf("the sources read are %q", got)
	}
	// Refusing every customisation refuses this vault's tools with them: they
	// arrive on a command line and are read as one.
	if slices.Contains(argv, "--safe-mode") {
		t.Error("safe mode takes this vault's own tools away")
	}
}

// The tools of this vault survive whatever refuses the machine's configuration.
// An agent that cannot reach the vault answers from what the model already
// knows, and says the vault was missing after the answer.
func TestThisVaultsToolsSurviveWhatIsRefused(t *testing.T) {
	for _, own := range []bool{false, true} {
		argv := recordedWith(t, func(a *claudecode.Agent) { a.ReadsHooksAndSkills = own })

		if slices.Contains(argv, "--safe-mode") {
			t.Errorf("own=%v: safe mode disables MCP servers, this vault's included", own)
		}
		if !slices.Contains(argv, "--mcp-config") || !slices.Contains(argv, "--strict-mcp-config") {
			t.Errorf("own=%v: the vault is not the server it was given: %q", own, argv)
		}
	}
}

// Asked for the person's own configuration, only theirs is read. A vault arrives
// from elsewhere, and a settings file inside one is a vault naming commands for
// this machine to run.
func TestAVaultsOwnConfigurationIsNeverRead(t *testing.T) {
	argv := recordedWith(t, func(a *claudecode.Agent) { a.ReadsHooksAndSkills = true })

	if slices.Contains(argv, "--safe-mode") {
		t.Error("the person asked for their own configuration and got none")
	}
	at := slices.Index(argv, "--setting-sources")
	if at < 0 {
		t.Fatalf("every source is read, a vault's included: %q", argv)
	}
	if got := argv[at+1]; got != "user" {
		t.Errorf("the sources read are %q", got)
	}
}

// No agent this window started outlives it, and a person has several
// conversations open at once. Each child is put in a process group of its own,
// so nothing that ends this process reaches it, and one still answering goes on
// writing to the vault with nobody watching.
func TestClosingEndsEveryAgentThatIsStillAnswering(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	// Says one thing and then waits, the way an agent between turns does.
	body := `#!/bin/sh
asked=$2
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
echo $$ > "` + dir + `/pid-$asked"
sleep 120
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	asked := map[string]string{"one": "left", "two": "right"}
	works := map[string]port.Work{}
	for conversation, question := range asked {
		work, err := claude.Take(context.Background(),
			port.Task{Asked: question, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		works[conversation] = work
	}

	pids := map[string]int{}
	for conversation, question := range asked {
		pid := 0
		for range 200 {
			raw, err := os.ReadFile(filepath.Join(dir, "pid-"+question))
			if err == nil {
				if pid, err = strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		if pid == 0 {
			t.Fatalf("the agent of conversation %q never started", conversation)
		}
		if err := syscall.Kill(pid, 0); err != nil {
			t.Fatalf("the agent of conversation %q is not running: %v", conversation, err)
		}
		pids[conversation] = pid
	}

	if err := claude.Close(); err != nil {
		t.Fatal(err)
	}

	// A process this one started stays visible until it is waited for, so what
	// says it is over is that a signal no longer reaches it.
	for conversation, pid := range pids {
		gone := false
		for range 200 {
			if err := syscall.Kill(pid, 0); err != nil {
				gone = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !gone {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Errorf("the agent of conversation %q outlived the window that started it", conversation)
		}
	}

	for _, work := range works {
		if _, taking := <-work.Steps(); taking {
			// Draining what was said before the close is fine; what must not
			// happen is the work going on.
			for range work.Steps() {
			}
		}
	}

	// A window that has closed does not start another.
	if _, err := claude.Take(context.Background(), port.Task{Asked: "again"}); err == nil {
		t.Error("an agent was started after the window closed")
	}
}

// A person keeps several conversations open at once, and each goes on in the
// one it was in.
//
// The script says which session it is on, named after what it was asked, and
// writes down what it was started with.
func TestEachConversationGoesOnInItsOwn(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := `#!/bin/sh
asked=$2
for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/argv-$asked"
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
echo '{"type":"result","subtype":"success","is_error":false}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	asks := func(asked, conversation string) {
		t.Helper()
		work, err := claude.Take(t.Context(), port.Task{Asked: asked, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		heard(t, work)
	}

	asks("left", "one")
	asks("right", "two")

	// A first question has no session behind it to go on with.
	for conversation, asked := range map[string]string{"one": "left", "two": "right"} {
		if session := resumed(argvOf(t, dir, asked)); session != "" {
			t.Errorf("the first question of conversation %q went on with %q, want none",
				conversation, session)
		}
	}

	asks("left-again", "one")
	asks("right-again", "two")

	if session := resumed(argvOf(t, dir, "left-again")); session != "s-left" {
		t.Errorf("conversation %q went on with %q, want %q", "one", session, "s-left")
	}
	if session := resumed(argvOf(t, dir, "right-again")); session != "s-right" {
		t.Errorf("conversation %q went on with %q, want %q", "two", session, "s-right")
	}
	if session := claude.Carrying("one"); session != "s-left-again" {
		t.Errorf("conversation %q is carrying %q, want %q", "one", session, "s-left-again")
	}
}

// Both are started before either has finished, the way two panels answering at
// once are.
func TestConversationsAnsweringAtOnceKeepTheirOwn(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	// Waits for the other to have started, so that neither finishes alone.
	body := `#!/bin/sh
asked=$2
touch "` + dir + `/started-$asked"
for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/argv-$asked"
until [ -f "` + dir + `/started-left" ] && [ -f "` + dir + `/started-right" ]; do sleep 0.01; done
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
echo '{"type":"result","subtype":"success","is_error":false}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	takes := func(asked, conversation string) port.Work {
		t.Helper()
		work, err := claude.Take(t.Context(), port.Task{Asked: asked, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		return work
	}

	left, right := takes("left", "one"), takes("right", "two")
	heard(t, left)
	heard(t, right)

	if session := claude.Carrying("one"); session != "s-left" {
		t.Errorf("conversation %q is carrying %q, want %q", "one", session, "s-left")
	}
	if session := claude.Carrying("two"); session != "s-right" {
		t.Errorf("conversation %q is carrying %q, want %q", "two", session, "s-right")
	}
}

// A question asked in no conversation is answered on its own: nothing it was
// told carries into the next one asked the same way.
func TestAQuestionInNoConversationCarriesNothing(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := `#!/bin/sh
asked=$2
for a in "$@"; do printf '%s\n' "$a"; done > "` + dir + `/argv-$asked"
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
echo '{"type":"result","subtype":"success","is_error":false}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	asks := func(asked string) {
		t.Helper()
		work, err := claude.Take(t.Context(), port.Task{Asked: asked})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		heard(t, work)
	}

	asks("first")
	asks("second")

	if session := resumed(argvOf(t, dir, "second")); session != "" {
		t.Errorf("a question in no conversation went on with %q, want none", session)
	}
	if session := claude.Carrying(""); session != "" {
		t.Errorf("no conversation is carrying %q, want none", session)
	}
}

// Stopping one conversation leaves every other where it was: a person closing
// one tab is still owed the answer in the next.
func TestStoppingOneConversationLeavesAnotherAnswering(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	// Says one thing, waits to be let on, and says the rest.
	body := `#!/bin/sh
asked=$2
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
printf '{"type":"assistant","message":{"content":[{"type":"text","text":"first %s"}]}}\n' "$asked"
until [ -f "` + dir + `/on-$asked" ]; do sleep 0.01; done
printf '{"type":"assistant","message":{"content":[{"type":"text","text":"second %s"}]}}\n' "$asked"
echo '{"type":"result","subtype":"success","is_error":false}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	takes := func(asked, conversation string) port.Work {
		t.Helper()
		work, err := claude.Take(context.Background(), port.Task{Asked: asked, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		return work
	}

	left, right := takes("left", "one"), takes("right", "two")

	// Both are mid-answer: each has said its first piece and neither is done.
	if step := <-left.Steps(); step.Text != "first left" {
		t.Fatalf("one conversation said %+v, want %q", step, "first left")
	}
	if step := <-right.Steps(); step.Text != "first right" {
		t.Fatalf("the other said %+v, want %q", step, "first right")
	}

	if err := left.Stop(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "on-right"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var said []string
	for step := range right.Steps() {
		if step.Kind == port.StepSaying {
			said = append(said, step.Text)
		}
	}
	if !slices.Contains(said, "second right") {
		t.Errorf("the conversation left running said %q, want %q among it", said, "second right")
	}
	if session := claude.Carrying("two"); session != "s-right" {
		t.Errorf("conversation %q is carrying %q, want %q", "two", session, "s-right")
	}
}

// argvOf is what the run answering one question was started with.
func argvOf(t *testing.T, dir, asked string) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dir, "argv-"+asked))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

// resumed is the session a run was told to go on with, empty when it was told
// none.
func resumed(argv []string) string {
	at := slices.Index(argv, "--resume")
	if at < 0 || at+1 >= len(argv) {
		return ""
	}
	return argv[at+1]
}

// A conversation the person closed is over, and the session it was on is let
// go of. Every other conversation is where it was.
func TestFinishingAConversationLetsGoOfWhatItWasOn(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := `#!/bin/sh
asked=$2
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
echo '{"type":"result","subtype":"success","is_error":false}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	asks := func(asked, conversation string) {
		t.Helper()
		work, err := claude.Take(t.Context(), port.Task{Asked: asked, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		heard(t, work)
	}

	asks("left", "one")
	asks("right", "two")

	if err := claude.Finish(t.Context(), "one"); err != nil {
		t.Fatal(err)
	}

	if session := claude.Carrying("one"); session != "" {
		t.Errorf("a conversation that is over is carrying %q, want none", session)
	}
	if session := claude.Carrying("two"); session != "s-right" {
		t.Errorf("conversation %q is carrying %q, want %q", "two", session, "s-right")
	}
}

// A person closing a tab mid-answer is the ordinary way a conversation ends.
// What was still answering in it goes with it, and the tab beside it is still
// owed its answer.
func TestFinishingAConversationEndsWhatIsStillAnsweringInIt(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	// Says one thing, writes down where it is, and waits the way an agent
	// between turns does.
	body := `#!/bin/sh
asked=$2
printf '{"type":"system","subtype":"init","session_id":"s-%s"}\n' "$asked"
printf '{"type":"assistant","message":{"content":[{"type":"text","text":"first %s"}]}}\n' "$asked"
echo $$ > "` + dir + `/pid-$asked"
sleep 120
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	}
	takes := func(asked, conversation string) port.Work {
		t.Helper()
		work, err := claude.Take(context.Background(), port.Task{Asked: asked, Conversation: conversation})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.Stop() })
		return work
	}

	left, right := takes("left", "one"), takes("right", "two")

	// Both are mid-answer: each has said its first piece and neither is done.
	if step := <-left.Steps(); step.Text != "first left" {
		t.Fatalf("one conversation said %+v, want %q", step, "first left")
	}
	if step := <-right.Steps(); step.Text != "first right" {
		t.Fatalf("the other said %+v, want %q", step, "first right")
	}
	closing, answering := pidOf(t, dir, "left"), pidOf(t, dir, "right")

	if err := claude.Finish(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}

	if !ended(closing) {
		syscall.Kill(closing, syscall.SIGKILL)
		t.Error("the agent of a conversation that is over is still running")
	}
	for range left.Steps() {
	}
	if err := syscall.Kill(answering, 0); err != nil {
		t.Errorf("the conversation left open stopped answering: %v", err)
	}
	if session := claude.Carrying("two"); session != "s-right" {
		t.Errorf("conversation %q is carrying %q, want %q", "two", session, "s-right")
	}
}

// pidOf is the process a run is, once it has written down where it is.
func pidOf(t *testing.T, dir, asked string) int {
	t.Helper()

	for range 200 {
		raw, err := os.ReadFile(filepath.Join(dir, "pid-"+asked))
		if err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the agent answering %q never said where it is", asked)
	return 0
}

// ended reports whether a process is over. One this process started stays
// visible until it is waited for, so what says it is over is that a signal no
// longer reaches it.
func ended(pid int) bool {
	for range 200 {
		if err := syscall.Kill(pid, 0); err != nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
