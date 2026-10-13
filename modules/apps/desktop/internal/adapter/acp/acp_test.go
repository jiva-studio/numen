package acp_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/acp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestAgent_FinishEmpty(t *testing.T) {
	agent := &acp.Agent{Program: "antigravity"}
	if err := agent.EndConversation(t.Context(), ""); err != nil {
		t.Errorf("unexpected error on empty finish: %v", err)
	}
}

func TestAgent_TakeWithoutTools(t *testing.T) {
	agent := &acp.Agent{Program: "antigravity"}
	_, err := agent.StartRun(t.Context(), port.Task{Question: "hello"})
	if err == nil {
		t.Errorf("expected error when no tools provided")
	}
}

// makeCodex is a codex agent whose program writes its arguments down, prints five finished tool
// calls and then waits to be killed.
func makeCodex(t *testing.T) (agent *acp.Agent, argv string) {
	t.Helper()

	dir := t.TempDir()
	argv = filepath.Join(dir, "argv")
	script := filepath.Join(dir, "codex")
	body := "#!/bin/sh\n" +
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + argv + "\n" +
		"for i in 1 2 3 4 5; do\n" +
		"  echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"mcp_tool_call\"}}'\n" +
		"done\n" +
		"sleep 60\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	agent = acp.New(acp.Options{
		Program: "codex",
		Root:    dir,
		Tools:   acp.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	})
	agent.Command = []string{script}
	t.Cleanup(func() { _ = agent.Close() })
	return agent, argv
}

// readSteps drains a run, failing when it does not end.
func readSteps(t *testing.T, run port.Run) []port.Step {
	t.Helper()

	var steps []port.Step
	deadline := time.After(10 * time.Second)
	for {
		select {
		case step, ok := <-run.GetSteps():
			if !ok {
				return steps
			}
			steps = append(steps, step)
		case <-deadline:
			t.Fatalf("the run did not end: %+v", steps)
		}
	}
}

// The task's step limit stops the agent after that many finished tool calls.
func TestMaxStepsStopsTheAgent(t *testing.T) {
	agent, _ := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q", MaxSteps: 2})
	if err != nil {
		t.Fatal(err)
	}
	steps := readSteps(t, run)

	var done int
	for _, step := range steps {
		if step.Kind == domain.StepToolDone {
			done++
		}
	}
	if done != 2 {
		t.Errorf("%d tool calls finished, want 2", done)
	}
	last := steps[len(steps)-1]
	if last.Kind != domain.StepStop || !strings.Contains(last.Detail, "2 steps") {
		t.Errorf("last step is %+v, want a stop naming the limit", last)
	}
}

// A task naming no limit runs to the end of what the agent prints.
func TestNoMaxStepsLetsTheAgentFinishItsOutput(t *testing.T) {
	agent, _ := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	var done int
	for step := range run.GetSteps() {
		if step.Kind == domain.StepToolDone {
			done++
		}
		if done == 5 {
			break
		}
	}
	if done != 5 {
		t.Errorf("%d tool calls finished, want 5", done)
	}
	_ = run.Stop()
}

// The instructions of the task go to the agent ahead of the question.
func TestInstructionsAreGivenAheadOfTheQuestion(t *testing.T) {
	agent, argv := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "what is here?", Instructions: "Call a note by its title.", MaxSteps: 1})
	if err != nil {
		t.Fatal(err)
	}
	readSteps(t, run)

	raw, err := os.ReadFile(argv)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if got, want := strings.Join(lines[len(lines)-2:], "\n"), "Call a note by its title.\nwhat is here?"; got != want {
		t.Errorf("the agent was asked %q, want %q", got, want)
	}
}

// isolate points the home, cache, config and temporary folders at fresh ones. It returns the
// home and the temporary folder.
func isolate(t *testing.T) (home, tmp string) {
	t.Helper()

	home = t.TempDir()
	tmp = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("TMPDIR", tmp)
	return home, tmp
}

// recorder is a program that writes down what it was given: its arguments, folder, token, a
// marker of the environment, its home, and the first three lines of its input.
const recorder = `for a in "$@"; do printf '%s\n' "$a"; done > $OUT/argv
pwd > $OUT/cwd
printf '%s' "$NUMEN_TOKEN" > $OUT/token
printf '%s' "$NUMEN_MARKER" > $OUT/marker
printf '%s' "$HOME" > $OUT/home
head -n 3 > $OUT/input
`

// makeProgram writes a program called binary with the body, which finds its own folder in $OUT.
// It returns that folder.
func makeProgram(t *testing.T, binary, body string) (out string) {
	t.Helper()

	out = t.TempDir()
	script := "#!/bin/sh\nOUT=" + out + "\n" + body
	if err := os.WriteFile(filepath.Join(out, binary), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return out
}

// makeAgent is an agent set to program that starts the program binary made of body. Its root is
// the folder the program writes to.
func makeAgent(t *testing.T, program, binary, body string) (agent *acp.Agent, out string) {
	t.Helper()

	out = makeProgram(t, binary, body)
	agent = acp.New(acp.Options{
		Program: program,
		Root:    out,
		Tools:   acp.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	})
	agent.Command = []string{filepath.Join(out, binary)}
	t.Cleanup(func() { _ = agent.Close() })
	return agent, out
}

// runTask asks the agent and drains the run.
func runTask(t *testing.T, agent *acp.Agent, task port.Task) []port.Step {
	t.Helper()

	run, err := agent.StartRun(t.Context(), task)
	if err != nil {
		t.Fatal(err)
	}
	return readSteps(t, run)
}

// readNote is what the program wrote to a file of its folder.
func readNote(t *testing.T, out, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(out, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// readNames is the names a program listed in a file, in order.
func readNames(t *testing.T, out, name string) string {
	t.Helper()

	names := strings.Fields(readNote(t, out, name))
	sort.Strings(names)
	return strings.Join(names, " ")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// isEmptyEventually is whether the folder is empty within a few seconds: the home made for a
// program is removed as its run ends.
func isEmptyEventually(t *testing.T, dir string) bool {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestTheProgramIsStartedTheWayItsKindIsAsked(t *testing.T) {
	cases := []struct {
		name, program, binary string
		inArgv, inInput       string
	}{
		{"antigravity named by the agent", "antigravity", "tool", "--output-format", ""},
		{"antigravity named by agy", "other", "agy", "--output-format", ""},
		{"antigravity named by its binary", "other", "antigravity", "--output-format", ""},
		{"codex named by the agent", "codex", "tool", "--json", ""},
		{"codex named by its binary", "other", "codex", "--json", ""},
		{"any other program speaks JSON-RPC", "other", "tool", "", `"method":"initialize"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isolate(t)
			agent, out := makeAgent(t, c.program, c.binary, recorder)

			runTask(t, agent, port.Task{Question: "hello"})
			argv, input := readNote(t, out, "argv"), readNote(t, out, "input")

			if c.inArgv == "" && argv != "" {
				t.Errorf("a JSON-RPC program was given arguments: %q", argv)
			}
			if !strings.Contains(argv, c.inArgv) {
				t.Errorf("arguments %q lack %q", argv, c.inArgv)
			}
			if c.inInput == "" && input != "" {
				t.Errorf("a program that is not asked over JSON-RPC read %q", input)
			}
			if !strings.Contains(input, c.inInput) {
				t.Errorf("input %q lacks %q", input, c.inInput)
			}
		})
	}
}

func TestAnAgentWithoutAToolURLOrATokenIsRefused(t *testing.T) {
	for name, tools := range map[string]acp.Endpoint{
		"no URL":   {Token: "let-me-in"},
		"no token": {URL: "http://127.0.0.1:7717/mcp"},
	} {
		t.Run(name, func(t *testing.T) {
			isolate(t)
			agent, _ := makeAgent(t, "codex", "codex", recorder)
			agent.Tools = tools

			if _, err := agent.StartRun(t.Context(), port.Task{Question: "hello"}); err == nil {
				t.Error("an agent that cannot reach the tools was started")
			}
		})
	}
}

func TestTheProgramRunsInTheRootWithTheEnvironmentAndTheToken(t *testing.T) {
	isolate(t)
	t.Setenv("NUMEN_MARKER", "kept")
	agent, out := makeAgent(t, "codex", "codex", recorder)

	runTask(t, agent, port.Task{Question: "hello"})

	wantDir, err := filepath.EvalSymlinks(out)
	if err != nil {
		t.Fatal(err)
	}
	gotDir, err := filepath.EvalSymlinks(strings.TrimSpace(readNote(t, out, "cwd")))
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != wantDir {
		t.Errorf("the program ran in %q, want %q", gotDir, wantDir)
	}
	if got := readNote(t, out, "marker"); got != "kept" {
		t.Errorf("the program saw the marker %q, want the one of the environment", got)
	}
	if got := readNote(t, out, "token"); got != "let-me-in" {
		t.Errorf("the program was given the token %q", got)
	}
}

func TestTheArgumentsOfTheCommandComeFirst(t *testing.T) {
	isolate(t)
	agent, out := makeAgent(t, "other", "tool", recorder)
	agent.Command = append(agent.Command, "--extra", "--more")

	runTask(t, agent, port.Task{Question: "hello"})

	if got := readNote(t, out, "argv"); got != "--extra\n--more\n" {
		t.Errorf("the program was given %q", got)
	}
}

func TestWithoutACommandTheProgramOfTheKindIsFoundOnThePath(t *testing.T) {
	for program, binary := range map[string]string{"codex": "codex", "antigravity": "agy"} {
		t.Run(program, func(t *testing.T) {
			isolate(t)
			agent, out := makeAgent(t, program, binary, recorder)
			agent.Command = nil
			t.Setenv("PATH", out+string(os.PathListSeparator)+os.Getenv("PATH"))

			runTask(t, agent, port.Task{Question: "hello"})

			if got := readNote(t, out, "argv"); got == "" {
				t.Error("the program on the path was started without arguments")
			}
		})
	}
}

func TestTheModelIsAskedForOnlyWhereOneIsNamed(t *testing.T) {
	cases := []struct {
		name, agentModel, taskModel, want string
	}{
		{"the agent's", "big", "", "big"},
		{"the task's over the agent's", "big", "small", "small"},
		{"none", "", "", ""},
		{"the default", "default", "", ""},
		{"the default of the task", "big", "default", ""},
	}
	for _, kind := range []string{"antigravity", "codex"} {
		for _, c := range cases {
			t.Run(kind+" "+c.name, func(t *testing.T) {
				isolate(t)
				agent, out := makeAgent(t, kind, "tool", recorder)
				agent.Model = c.agentModel

				runTask(t, agent, port.Task{Question: "hello", Model: c.taskModel})

				lines := strings.Split(readNote(t, out, "argv"), "\n")
				got, want := "no model", c.want
				if want == "" {
					want = "no model"
				}
				for i, line := range lines {
					if line == "--model" && i+1 < len(lines) {
						got = lines[i+1]
					}
				}
				if got != want {
					t.Errorf("the model asked for is %q, want %q", got, want)
				}
			})
		}
	}
}

func TestAConversationCarriesOnInTheSessionItIsOn(t *testing.T) {
	isolate(t)
	agent, out := makeAgent(t, "antigravity", "tool", recorder+"echo '{\"conversation_id\":\"abc\"}'\n")

	runTask(t, agent, port.Task{Question: "one", Conversation: "talk"})
	if got := readNote(t, out, "argv"); strings.Contains(got, "--conversation") {
		t.Errorf("the first task was given a session: %q", got)
	}
	runTask(t, agent, port.Task{Question: "two", Conversation: "talk"})

	if got := readNote(t, out, "argv"); !strings.Contains(got, "--conversation\nabc\n") {
		t.Errorf("the second task was given %q, want the session of the first", got)
	}
}

func TestTheQuestionIsAskedAsItIsWhereNoInstructionsAreGiven(t *testing.T) {
	isolate(t)
	agent, out := makeAgent(t, "codex", "tool", recorder)

	runTask(t, agent, port.Task{Question: "what is here?"})

	lines := strings.Split(strings.TrimRight(readNote(t, out, "argv"), "\n"), "\n")
	if got := lines[len(lines)-1]; got != "what is here?" {
		t.Errorf("the agent was asked %q", got)
	}
}

func TestAJSONRPCProgramIsAskedOverItsInput(t *testing.T) {
	isolate(t)
	agent, out := makeAgent(t, "other", "tool", recorder)

	runTask(t, agent, port.Task{Question: "what is here?", Instructions: "Be brief.", Model: "big", Focus: "a.md"})

	type request struct {
		ID     int            `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	var requests []request
	lines := strings.Split(strings.TrimRight(readNote(t, out, "input"), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d requests were written, want 3: %q", len(lines), lines)
	}
	for _, line := range lines {
		var r request
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("%q is not a request: %v", line, err)
		}
		requests = append(requests, r)
	}
	for i, method := range []string{"initialize", "session/new", "session/prompt"} {
		if requests[i].ID != i+1 || requests[i].Method != method {
			t.Errorf("request %d is %d %q, want %d %q", i, requests[i].ID, requests[i].Method, i+1, method)
		}
	}

	session := requests[1].Params
	if session["cwd"] != out {
		t.Errorf("the session is opened in %v, want %v", session["cwd"], out)
	}
	server := session["mcpServers"].([]any)[0].(map[string]any)["transport"].(map[string]any)
	if server["url"] != "http://127.0.0.1:7717/mcp" {
		t.Errorf("the tools are at %v", server["url"])
	}
	if got := server["headers"].(map[string]any)["Authorization"]; got != "Bearer let-me-in" {
		t.Errorf("the tools are reached with %v", got)
	}

	prompt := requests[2].Params
	if prompt["model"] != "big" {
		t.Errorf("the model is %v", prompt["model"])
	}
	if got := prompt["prompt"].([]any)[0].(map[string]any)["text"]; got != "Be brief.\nwhat is here?" {
		t.Errorf("the prompt is %q", got)
	}
	if got := prompt["context"].(map[string]any)["focus"]; got != "a.md" {
		t.Errorf("the focus is %v", got)
	}
}

func TestBlankAndPlainLinesOfTheOutputAreSkipped(t *testing.T) {
	isolate(t)
	agent, _ := makeAgent(t, "codex", "tool", "echo\necho 'not json'\necho '{\"content\":\"hello\"}'\n")

	steps := runTask(t, agent, port.Task{Question: "hello"})

	if len(steps) != 1 || steps[0].Text != "hello" {
		t.Errorf("the steps are %+v, want the one line that was said", steps)
	}
}

func TestAProgramThatCannotBeStartedIsReported(t *testing.T) {
	_, tmp := isolate(t)
	agent, out := makeAgent(t, "antigravity", "tool", recorder)
	agent.Command = []string{filepath.Join(out, "missing")}

	_, err := agent.StartRun(t.Context(), port.Task{Question: "hello"})

	if err == nil || !strings.HasPrefix(err.Error(), "start ") || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the failure was reported as %v", err)
	}
	if !isEmptyEventually(t, tmp) {
		t.Error("the home made for the program was left behind")
	}
}

func TestAClosedAgentStartsNothing(t *testing.T) {
	_, tmp := isolate(t)
	agent, _ := makeAgent(t, "antigravity", "tool", recorder)
	_ = agent.Close()

	_, err := agent.StartRun(t.Context(), port.Task{Question: "hello"})

	if err == nil || !strings.Contains(err.Error(), "closing") {
		t.Errorf("a closed agent answered %v", err)
	}
	if !isEmptyEventually(t, tmp) {
		t.Error("the home made for the program was left behind")
	}
}

func TestStoppingAProgramStopsWhatItStarted(t *testing.T) {
	isolate(t)
	agent, out := makeAgent(t, "codex", "tool", `(sleep 1; touch $OUT/survived) &
for i in 1 2 3 4 5; do
  echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'
done
wait
`)

	runTask(t, agent, port.Task{Question: "q", MaxSteps: 1})
	time.Sleep(1500 * time.Millisecond)

	if _, err := os.Stat(filepath.Join(out, "survived")); err == nil {
		t.Error("a process the program started outlived the run")
	}
}

func TestSixtyFourStepsAreHeldWithoutBeingRead(t *testing.T) {
	isolate(t)
	body := ""
	for range 64 {
		body += "echo '{\"content\":\"x\"}'\n"
	}
	agent, _ := makeAgent(t, "codex", "tool", body)
	run, err := agent.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}

	stopped := make(chan struct{})
	go func() {
		_ = run.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Error("the run did not end with its steps unread")
		for range run.GetSteps() {
		}
	}
}

// homeDump writes down the home it is started in.
const homeDump = `H=$HOME
ls -A $H > $OUT/top
ls -A $H/.gemini > $OUT/gemini
ls -A $H/.gemini/config > $OUT/config
cat $H/keep > $OUT/keep
cat $H/.gemini/other > $OUT/other
cat $H/.gemini/config/mcp_config.json > $OUT/mcp
ls -ld $H/.gemini $H/.gemini/config $H/.gemini/config/mcp_config.json > $OUT/modes
[ -L $H/keep ] && [ -L $H/.gemini/other ] && echo yes > $OUT/linked
printf '%s' "$H" > $OUT/home
`

func TestAntigravityRunsInAHomeThatLinksToTheRealOne(t *testing.T) {
	home, tmp := isolate(t)
	writeFile(t, filepath.Join(home, "keep"), "kept")
	writeFile(t, filepath.Join(home, ".gemini", "other"), "other")
	writeFile(t, filepath.Join(home, ".gemini", "config", "extra.json"), "{}")
	writeFile(t, filepath.Join(home, ".gemini", "config", "mcp_config.json"),
		`{"mcpServers":{"numen":{"old":true},"theirs":{"command":"x"}}}`)
	agent, out := makeAgent(t, "antigravity", "tool", homeDump)

	runTask(t, agent, port.Task{Question: "hello"})

	if got := readNote(t, out, "home"); got == home || !strings.HasPrefix(got, tmp) {
		t.Errorf("the program ran in %q, want a home made in %q", got, tmp)
	}
	if got := readNames(t, out, "top"); got != ".gemini keep" {
		t.Errorf("the home holds %v", got)
	}
	if got := readNames(t, out, "gemini"); got != "config other" {
		t.Errorf(".gemini holds %v", got)
	}
	if got := readNames(t, out, "config"); got != "mcp_config.json" {
		t.Errorf("the configuration folder holds %v", got)
	}
	if readNote(t, out, "keep") != "kept" || readNote(t, out, "other") != "other" ||
		strings.TrimSpace(readNote(t, out, "linked")) != "yes" {
		t.Error("the entries of the real home are not linked")
	}

	modes := strings.Split(strings.TrimSpace(readNote(t, out, "modes")), "\n")
	for i, want := range []string{"drwx------", "drwx------", "-rw-------"} {
		if len(modes[i]) < 10 || modes[i][:10] != want {
			t.Errorf("mode %d is %q, want %s", i, modes[i], want)
		}
	}

	raw := readNote(t, out, "mcp")
	if !strings.HasSuffix(raw, "}\n") {
		t.Errorf("the configuration does not end its line: %q", raw)
	}
	var config struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatal(err)
	}
	numen := config.MCPServers["numen"]
	if numen["serverUrl"] != "http://127.0.0.1:7717/mcp" || numen["old"] != nil {
		t.Errorf("the vault's tools are named %v", numen)
	}
	if got := numen["headers"].(map[string]any)["Authorization"]; got != "Bearer let-me-in" {
		t.Errorf("the tools are reached with %v", got)
	}
	if config.MCPServers["theirs"]["command"] != "x" {
		t.Errorf("the servers of the existing configuration were lost: %v", config.MCPServers)
	}

	if !isEmptyEventually(t, tmp) {
		t.Error("the home made for the program was left behind")
	}
}

func TestAnExistingConfigurationThatIsNotJSONIsLeftOut(t *testing.T) {
	home, _ := isolate(t)
	writeFile(t, filepath.Join(home, ".gemini", "config", "mcp_config.json"), "{not json")
	writeFile(t, filepath.Join(home, "keep"), "kept")
	writeFile(t, filepath.Join(home, ".gemini", "other"), "other")
	agent, out := makeAgent(t, "antigravity", "tool", homeDump)

	runTask(t, agent, port.Task{Question: "hello"})

	var config struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(readNote(t, out, "mcp")), &config); err != nil {
		t.Fatal(err)
	}
	if len(config.MCPServers) != 1 || config.MCPServers["numen"] == nil {
		t.Errorf("the servers are %v, want only the vault's", config.MCPServers)
	}
}

func TestAntigravityRunsInTheRealHomeWhereNoneCanBeMade(t *testing.T) {
	cases := map[string]func(t *testing.T, tmp string){
		"no temporary folder": func(t *testing.T, tmp string) { t.Setenv("TMPDIR", filepath.Join(tmp, "missing")) },
		"no home":             func(t *testing.T, _ string) { t.Setenv("HOME", "") },
	}
	for name, spoil := range cases {
		t.Run(name, func(t *testing.T) {
			_, tmp := isolate(t)
			agent, out := makeAgent(t, "antigravity", "tool", recorder)
			spoil(t, tmp)

			runTask(t, agent, port.Task{Question: "hello"})

			if got := readNote(t, out, "argv"); !strings.Contains(got, "--output-format") {
				t.Errorf("the program was not started: %q", got)
			}
			if got := readNote(t, out, "home"); got != os.Getenv("HOME") {
				t.Errorf("the program ran in the home %q, want %q", got, os.Getenv("HOME"))
			}
			if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
				t.Errorf("a home was left behind: %v %v", entries, err)
			}
		})
	}
}

// Stopping a run ends it although nobody reads its steps and its buffer is full.
func TestStopEndsARunWhoseStepsAreNotRead(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	body := "#!/bin/sh\n" +
		"for i in $(seq 1 200); do\n" +
		"  echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"mcp_tool_call\"}}'\n" +
		"done\n" +
		"sleep 60\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	agent := acp.New(acp.Options{
		Program: "codex",
		Root:    dir,
		Tools:   acp.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	})
	agent.Command = []string{script}
	t.Cleanup(func() { _ = agent.Close() })

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	for len(run.GetSteps()) < cap(run.GetSteps()) {
		time.Sleep(10 * time.Millisecond)
	}

	stopped := make(chan struct{})
	go func() {
		_ = run.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return")
	}
}
