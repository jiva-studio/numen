package service

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/toolcall"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

var testWords = map[string]toolcall.Declaration{
	"note_edit": {Title: "Edit a note", Kind: "edit", About: "path", Match: "stood", Text: "becomes"},
	"note_read": {Title: "Read notes", Kind: "read", About: "paths", Element: "path"},
}

func parseLines(t *testing.T, words map[string]toolcall.Declaration, lines ...string) ([]*numenv1.AskAgentResponse, string) {
	t.Helper()
	var got []*numenv1.AskAgentResponse
	send := func(r *numenv1.AskAgentResponse) error {
		got = append(got, r)
		return nil
	}
	tick := time.Unix(0, 0)
	reason := parseStreamWithClock(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n"), send, words, nil,
		func() time.Time { tick = tick.Add(time.Second); return tick })
	return got, reason
}

func toolCalls(steps []*numenv1.AskAgentResponse) []*numenv1.ToolCall {
	var calls []*numenv1.ToolCall
	for _, s := range steps {
		if c := s.GetToolCall(); c != nil {
			calls = append(calls, c)
		}
	}
	return calls
}

const startEdit = `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"toolu_01","name":"mcp__numen__note_edit"}}}`

func delta(partial string) string {
	return `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":` +
		quote(partial) + `}}}`
}

func quote(text string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text) + `"`
}

func TestADeclaredToolIsShownByItsTitleAndEffect(t *testing.T) {
	steps, _ := parseLines(t, testWords, startEdit, delta(`{"path":"Note.md","stood":"a"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`)

	calls := toolCalls(steps)
	last := calls[len(calls)-1]
	if last.GetTool() != "Edit a note" || last.GetKind() != "edit" || last.GetAbout() != "Note.md" || last.GetPath() != "Note.md" {
		t.Fatalf("unexpected call: %+v", last)
	}
}

func TestEveryReportOfOneCallCarriesTheNameTheAgentGaveIt(t *testing.T) {
	steps, _ := parseLines(t, testWords, startEdit, delta(`{"path":"Note.md"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`)

	for _, c := range toolCalls(steps) {
		if c.GetCall() != "toolu_01" {
			t.Errorf("call id %q", c.GetCall())
		}
	}
}

func TestAnUndeclaredToolIsNamedWithoutItsServerPrefix(t *testing.T) {
	steps, _ := parseLines(t, nil,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c","name":"mcp__numen__other_tool"}}}`)

	calls := toolCalls(steps)
	if len(calls) != 1 || calls[0].GetTool() != "other_tool" || calls[0].GetKind() != "" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestACollectionCallIsAboutItsFirstElementAndTheRest(t *testing.T) {
	steps, _ := parseLines(t, testWords,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c","name":"mcp__numen__note_read","input":{"paths":[{"path":"A.md"},{"path":"B.md"},{"path":"C.md"}]}}]}}`)

	calls := toolCalls(steps)
	if len(calls) != 1 || calls[0].GetAbout() != "A.md and 2 more" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestACallNamingARunOfTextCarriesItsSpan(t *testing.T) {
	steps, _ := parseLines(t, testWords,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c","name":"mcp__numen__note_edit","input":{"path":"N.md","start":4,"length":6}}]}}`)

	span := toolCalls(steps)[0].GetSpan()
	if span.GetFrom() != 4 || span.GetTo() != 10 {
		t.Fatalf("span %+v", span)
	}
}

func editProgress(steps []*numenv1.AskAgentResponse) []*numenv1.EditProgress {
	var out []*numenv1.EditProgress
	for _, s := range steps {
		if e := s.GetEditProgress(); e != nil {
			out = append(out, e)
		}
	}
	return out
}

func TestNoChangeIsReportedBeforeItsReplacementBegins(t *testing.T) {
	steps, _ := parseLines(t, testWords, startEdit, delta(`{"path":"Note.md",`), delta(`"stood":"A hedgehog`), delta(` is named"`))

	if got := editProgress(steps); len(got) != 0 {
		t.Fatalf("reported early: %+v", got)
	}
}

func TestAChangeIsReportedAsItsReplacementArrives(t *testing.T) {
	steps, _ := parseLines(t, testWords, startEdit,
		delta(`{"path":"Note.md","stood":"A hedgehog",`), delta(`"becomes":"An axe`), delta(`, two-bladed"}`))

	got := editProgress(steps)
	if len(got) != 2 {
		t.Fatalf("expected two reports, got %+v", got)
	}
	if got[0].GetPath() != "Note.md" || got[0].GetMatch() != "A hedgehog" || got[0].GetText() != "An axe" || got[0].GetCall() != "toolu_01" {
		t.Errorf("first report %+v", got[0])
	}
	if got[1].GetText() != "An axe, two-bladed" {
		t.Errorf("second report %+v", got[1])
	}
}

func TestChangesCloserThanTheIntervalAreReportedOnce(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	p := &streamParser{
		send:  func(r *numenv1.AskAgentResponse) error { got = append(got, r); return nil },
		words: testWords,
		now:   func() time.Time { return time.Unix(100, 0) },
	}
	p.tool, p.callID = "mcp__numen__note_edit", "c"
	p.arguments.WriteString(`{"path":"N.md","stood":"x","becomes":"y`)
	p.reportEdit()
	p.reportEdit()

	if len(editProgress(got)) != 1 {
		t.Fatalf("expected one report, got %d", len(editProgress(got)))
	}
}

func TestParseLines_VaultMcpFailure(t *testing.T) {
	cases := map[string]string{
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"numen","status":"failed"}]}`:         "the agent could not reach this vault",
		`{"type":"system","subtype":"init","mcp_servers":[]}`:                                           "the agent was started without this vault",
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"numen","status":"connected"}]}`:      "",
		`{"type":"system","subtype":"init"}`:                                                            "",
		`{"type":"system","subtype":"init","mcp_server_errors":[{"name":"numen","message":"rejected"}]}`: "the agent was not given this vault: rejected",
	}
	for line, want := range cases {
		if _, reason := parseLines(t, nil, line); reason != want {
			t.Errorf("%s: got %q want %q", line, reason, want)
		}
	}
}

func TestParseLines_ErrorResultFormatting(t *testing.T) {
	cases := map[string]string{
		`{"type":"result","is_error":true,"subtype":"error_max_turns"}`:                            "the agent reached the limit of steps without finishing",
		`{"type":"result","is_error":true,"subtype":"error_context_length_exceeded"}`:              "the conversation is too long for the model",
		`{"type":"result","is_error":true,"subtype":"error_something_new"}`:                        "something new",
		`{"type":"result","is_error":true,"subtype":"error_max_turns","result":"error_max_turns"}`: "the agent reached the limit of steps without finishing",
		`{"type":"result","is_error":true,"result":"overloaded, try later"}`:                       "overloaded, try later",
		`{"type":"result","is_error":true}`:                                                        "the agent stopped without finishing",
	}
	for line, want := range cases {
		if _, reason := parseLines(t, nil, line); reason != want {
			t.Errorf("%s: got %q want %q", line, reason, want)
		}
	}
}

func TestParseLines_FirstErrorPriority(t *testing.T) {
	_, reason := parseLines(t, nil,
		`{"type":"result","is_error":true,"subtype":"error_rate_limit"}`,
		`{"type":"result","is_error":true,"subtype":"error_overloaded"}`)
	if reason != "the rate limit was reached" {
		t.Fatalf("got %q", reason)
	}
}

func TestBuildClaudeArguments_SettingSources(t *testing.T) {
	args := buildClaudeArguments(claudeArguments{configuration: "cfg.json", tools: []string{"note_read"}})
	assertPair(t, args, "--setting-sources", "")
	args = buildClaudeArguments(claudeArguments{configuration: "cfg.json", tools: []string{"note_read"}, mayReadHooks: true})
	assertPair(t, args, "--setting-sources", "user")
}

func TestTheAllowanceNamesTheSearchAndTheTasksTools(t *testing.T) {
	args := buildClaudeArguments(claudeArguments{configuration: "cfg.json", tools: []string{"note_read", "note_edit"}})
	assertPair(t, args, "--allowedTools", "WebSearch,mcp__numen__note_edit,mcp__numen__note_read")
	assertPair(t, args, "--tools", "WebSearch")
}

func TestTheDefaultNumberOfTurnsIsOneHundred(t *testing.T) {
	assertPair(t, buildClaudeArguments(claudeArguments{tools: []string{"note_read"}}), "--max-turns", "100")
	assertPair(t, buildClaudeArguments(claudeArguments{turns: 7, tools: []string{"note_read"}}), "--max-turns", "7")
}

func assertPair(t *testing.T, args []string, flag, want string) {
	t.Helper()
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			if args[i+1] != want {
				t.Errorf("%s = %q, want %q", flag, args[i+1], want)
			}
			return
		}
	}
	t.Errorf("%s absent from %v", flag, args)
}

func TestBuildEnvironment_SessionSanitization(t *testing.T) {
	got := buildEnvironment([]string{"PATH=/bin", "CLAUDECODE=1", "CLAUDE_CODE_SESSION_ID=x", "HOME=/h"})
	if strings.Join(got, ",") != "PATH=/bin,HOME=/h" {
		t.Fatalf("got %v", got)
	}
}

func TestAChangeIsLocatedByThePathArgumentAlone(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	p := &streamParser{
		send:  func(r *numenv1.AskAgentResponse) error { got = append(got, r); return nil },
		words: map[string]toolcall.Declaration{"note_edit": {About: "title", Match: "stood", Text: "becomes"}},
		now:   func() time.Time { return time.Unix(100, 0) },
	}
	p.tool, p.callID = "mcp__numen__note_edit", "c"
	p.arguments.WriteString(`{"title":"T","stood":"x","becomes":"y`)
	p.reportEdit()

	if len(editProgress(got)) != 0 {
		t.Fatalf("a title was reported as a path: %+v", editProgress(got))
	}
}

func TestStreamParserPreviewIntervalPacing(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	send := func(r *numenv1.AskAgentResponse) error {
		got = append(got, r)
		return nil
	}
	tick := time.Unix(0, 0)
	words := map[string]toolcall.Declaration{
		"note_edit": {About: "Edit a note", Match: "path", Text: "text"},
	}
	r := strings.NewReader(strings.Join([]string{
		startEdit,
		delta(`{"path":"a.md","text":"first"}`),
		delta(`{"path":"a.md","text":"second"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n") + "\n")

	parseStreamWithClock(context.Background(), r, send, words, nil, func() time.Time {
		tick = tick.Add(50 * time.Millisecond)
		return tick
	})

	edits := editProgress(got)
	if len(edits) != 2 {
		t.Fatalf("expected 2 edit progress steps spaced by 50ms, got %d", len(edits))
	}
}

func TestStreamParserInitSessionRecord(t *testing.T) {
	send := func(r *numenv1.AskAgentResponse) error {
		return nil
	}
	var recorded []string
	record := func(sess string) { recorded = append(recorded, sess) }

	r := strings.NewReader(`{"type":"system","subtype":"init","session_id":"sess-123"}` + "\n")
	parseStreamWithClock(context.Background(), r, send, nil, record, time.Now)
	if len(recorded) != 1 || recorded[0] != "sess-123" {
		t.Fatalf("expected one recorded session %q, got %v", "sess-123", recorded)
	}

	r2 := strings.NewReader(`{"type":"system","subtype":"init","session_id":""}` + "\n")
	parseStreamWithClock(context.Background(), r2, send, nil, record, time.Now)
	if len(recorded) != 1 {
		t.Fatalf("expected empty session to not be recorded, got %v", recorded)
	}
}

func TestStreamParserToolCallChunkAndReset(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	send := func(r *numenv1.AskAgentResponse) error {
		got = append(got, r)
		return nil
	}
	chunk200 := strings.Repeat("x", 200)
	r := strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c1","name":"mcp__numen__t1"}}}`,
		delta(`old_args`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c2","name":"mcp__numen__t2"}}}`,
		delta(chunk200),
		delta("1234567890"),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"","name":"mcp__numen__t3"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n") + "\n")

	parseStreamWithClock(context.Background(), r, send, nil, nil, time.Now)

	calls := toolCalls(got)
	var t2Calls []*numenv1.ToolCall
	for _, c := range calls {
		if c.GetTool() == "t2" {
			t2Calls = append(t2Calls, c)
		}
	}
	if len(t2Calls) != 3 {
		t.Fatalf("expected exactly 3 calls for t2, got %d (%+v)", len(t2Calls), t2Calls)
	}
	if t2Calls[1].GetLength() != 200 {
		t.Errorf("expected written 200 for chunk, got %d", t2Calls[1].GetLength())
	}
	var t3Calls []*numenv1.ToolCall
	for _, c := range calls {
		if c.GetTool() == "t3" {
			t3Calls = append(t3Calls, c)
		}
	}
	if len(t3Calls) == 0 || t3Calls[0].GetCall() != "" {
		t.Fatalf("expected empty call id for t3, got %+v", t3Calls)
	}
}

func TestStreamParserAssistantMessageIgnoredWhenStreaming(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	send := func(r *numenv1.AskAgentResponse) error {
		got = append(got, r)
		return nil
	}
	r := strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`,
	}, "\n") + "\n")

	parseStreamWithClock(context.Background(), r, send, nil, nil, time.Now)

	saids := 0
	for _, s := range got {
		if s.GetText() != "" {
			saids++
		}
	}
	if saids != 1 {
		t.Fatalf("expected assistant message to be ignored while streaming, got %d said steps", saids)
	}
}

func TestReportEditIgnoredWhenToolHasNoTextDeclaration(t *testing.T) {
	var got []*numenv1.AskAgentResponse
	send := func(r *numenv1.AskAgentResponse) error {
		got = append(got, r)
		return nil
	}
	words := map[string]toolcall.Declaration{
		"note_read": {About: "Read notes", Match: "path", Text: ""},
	}
	r := strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c1","name":"mcp__numen__note_read"}}}`,
		delta(`{"path":"a.md"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n") + "\n")

	parseStreamWithClock(context.Background(), r, send, words, nil, time.Now)

	for _, s := range got {
		if ep := s.GetEditProgress(); ep != nil {
			t.Fatalf("expected no edit progress when tool text is empty, got %+v", ep)
		}
	}
}

func TestTheLevelOfReasoningIsHandedToTheProgramAsItsEffort(t *testing.T) {
	for _, tools := range [][]string{{"note_read"}, nil} {
		args := buildClaudeArguments(claudeArguments{reasoning: "high", tools: tools})
		assertPair(t, args, "--effort", "high")
		if slices.Contains(buildClaudeArguments(claudeArguments{tools: tools}), "--effort") {
			t.Error("a run that names no level is given an effort")
		}
	}
}
