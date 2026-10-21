package settings_test

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

// read is the section as the file gives it over the defaults.
func read(t *testing.T, file string) settings.Agent {
	t.Helper()
	var held settings.Agent
	if err := json.Unmarshal([]byte(file), &held); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return held
}

// A settings file that names no agent names no agent. A file that leaves the name out leaves the
// choice to the default, and an empty name is what a person wrote.
func TestNamingNoAgentIsKept(t *testing.T) {
	var unmodified settings.Agent
	if !unmodified.IsNamed() || unmodified.GetUse() != "" {
		t.Fatalf("the defaults name %q", unmodified.GetUse())
	}

	held := read(t, `{"use": ""}`)
	if held.IsNamed() || held.GetUse() != "" {
		t.Errorf("a file naming no agent names %q", held.GetUse())
	}
	if got := held.GetProgram("claude").MaxSteps; got != settings.DefaultAgentSteps {
		t.Errorf("a program the file left out lost its defaults: %d", got)
	}

	if named := read(t, `{"use": "codex"}`); named.GetUse() != "codex" || !named.IsNamed() {
		t.Errorf("the file names %q", named.GetUse())
	}
	if again := read(t, `{"use": null}`); !again.IsNamed() || again.GetUse() != "" {
		t.Errorf("a null name names %q", again.GetUse())
	}
}

// What puts the tools on a port: an agent named for the panel, or a person asking for the port
// itself. An installation asking for neither opens none.
func TestWhatOpensThePort(t *testing.T) {
	var defaultAgent settings.Agent
	if defaultAgent.ShouldServeTools {
		t.Error("the defaults serve the tools to an agent nobody has configured")
	}

	for name, said := range map[string]struct {
		file      string
		isServing bool
	}{
		"the defaults":              {`{}`, true},
		"no agent named":            {`{"use": ""}`, false},
		"no agent, tools asked for": {`{"use": "", "serve_tools": true}`, true},
		"an agent named":            {`{"use": "claude"}`, true},
		"any program named":         {`{"use": "other"}`, true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := read(t, said.file).IsServingTools(); got != said.isServing {
				t.Errorf("the tools are served: %v", got)
			}
		})
	}
}

// A section a file leaves empty keeps the step limit the defaults give it, for each program alike.
func TestAnEmptySectionKeepsItsStepLimit(t *testing.T) {
	held := read(t, `{"claude": {}, "antigravity": {}, "codex": {}}`)
	for _, name := range []string{"claude", "antigravity", "codex"} {
		if got := held.GetProgram(name).MaxSteps; got != settings.DefaultAgentSteps {
			t.Errorf("%s may go to the model %d times", name, got)
		}
	}
}

// A section of the wrong type is a settings file that cannot be read.
func TestASectionOfTheWrongTypeIsRefused(t *testing.T) {
	for _, file := range []string{
		`{"claude": 5}`,
		`{"codex": "x"}`,
		`{"claude": {"max_steps": "many"}}`,
		`{"codex": {"command": 3}}`,
		`{"use": 5}`,
		`{"serve_tools": "yes"}`,
		`[]`,
	} {
		var held settings.Agent
		if err := json.Unmarshal([]byte(file), &held); err == nil {
			t.Errorf("%s was read", file)
		}
	}
}

// Every field a section names is taken over the defaults, for each program alike, whatever the
// program is called.
func TestTheFieldsASectionNamesAreTaken(t *testing.T) {
	held := read(t, `{
		"claude": {"command": ["c", "--x"], "model": "opus", "max_steps": 7, "reads_hooks_and_skills": true, "extra": [1,2]},
		"anything": {"command": ["x", "y"], "model": "gpt", "max_steps": 9, "reads_hooks_and_skills": true, "extra": [1,2]}
	}`)

	want := map[string]settings.AgentProgram{
		"claude":   {Command: []string{"c", "--x"}, Model: "opus", MaxSteps: 7, Options: map[string]string{"reads_hooks_and_skills": "true", "extra": "[1,2]"}},
		"anything": {Command: []string{"x", "y"}, Model: "gpt", MaxSteps: 9, Options: map[string]string{"reads_hooks_and_skills": "true", "extra": "[1,2]"}},
	}
	for name, program := range want {
		got := held.GetProgram(name)
		if !slices.Equal(got.Command, program.Command) || got.Model != program.Model ||
			got.MaxSteps != program.MaxSteps || !maps.Equal(got.Options, program.Options) {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

// A file read twice into one section keeps what the first said and the second left out.
func TestASectionReadTwiceKeepsWhatTheFirstSaid(t *testing.T) {
	var held settings.Agent
	for _, file := range []string{`{"claude": {"model": "opus"}}`, `{"claude": {"max_steps": 5}}`} {
		if err := json.Unmarshal([]byte(file), &held); err != nil {
			t.Fatal(err)
		}
	}
	if got := held.GetProgram("claude"); got.Model != "opus" || got.MaxSteps != 5 {
		t.Errorf("got %+v", got)
	}
}

// The command lines the settings give are those of the programs that set one.
func TestTheCommandsAreThoseTheSettingsSet(t *testing.T) {
	held := read(t, `{"claude": {"command": ["/c"]}, "codex": {"model": "m"}, "agy": {"command": []}}`)
	got := held.GetCommands()
	if len(got) != 1 || !slices.Equal(got["claude"], []string{"/c"}) {
		t.Errorf("got %v", got)
	}
}

// The section is written with the programs beside the two keys of it, and a choice left to the
// default is written as null, so that it reads back the same.
func TestTheSectionIsWrittenAsItIsRead(t *testing.T) {
	file := `{"use":"claude","serve_tools":true,"claude":{"command":["/c"],"model":"opus","reasoning":"high","max_steps":7,"reads_hooks_and_skills":true}}`
	written, err := json.Marshal(read(t, file))
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(written, &back); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(file), &want); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(back, want) {
		t.Errorf("wrote %s, read %s", written, file)
	}

	defaults, err := json.Marshal(settings.Agent{})
	if err != nil {
		t.Fatal(err)
	}
	if string(defaults) != `{"serve_tools":false,"use":null}` {
		t.Errorf("the defaults are written as %s", defaults)
	}
	none, err := json.Marshal(settings.Agent{Use: settings.UseProgram("")})
	if err != nil {
		t.Fatal(err)
	}
	if string(none) != `{"serve_tools":false,"use":""}` {
		t.Errorf("no agent is written as %s", none)
	}
}

func jsonEqual(a, b map[string]any) bool {
	left, errLeft := json.Marshal(a)
	right, errRight := json.Marshal(b)
	return errLeft == nil && errRight == nil && string(left) == string(right)
}
