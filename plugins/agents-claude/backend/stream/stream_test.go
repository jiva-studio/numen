package stream

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

func getTestWords() map[string]toolcall.Declaration {
	return map[string]toolcall.Declaration{
		"note_edit": {Title: "Edit a note", Kind: "edit", About: "path", Match: "stood", Text: "becomes"},
		"note_read": {Title: "Read notes", Kind: "read", About: "paths", Element: "path"},
	}
}

// steppingClock advances by step on every reading.
type steppingClock struct {
	tick time.Time
	step time.Duration
}

func (c *steppingClock) Now() time.Time {
	c.tick = c.tick.Add(c.step)
	return c.tick
}

func parseStreamWithClock(
	ctx context.Context,
	r io.Reader,
	send func(*agentrun.Response) error,
	words map[string]toolcall.Declaration,
	record func(string),
	clk Clock,
) string {
	return Parse(ctx, r, send, Options{Words: words, Record: record, Clock: clk, ExpectsVault: true})
}

func parseLines(t *testing.T, words map[string]toolcall.Declaration, lines ...string) ([]*agentrun.Response, string) {
	t.Helper()
	var got []*agentrun.Response
	send := func(r *agentrun.Response) error {
		got = append(got, r)
		return nil
	}
	reason := parseStreamWithClock(t.Context(), strings.NewReader(strings.Join(lines, "\n")+"\n"), send, words, nil,
		&steppingClock{tick: time.Unix(0, 0), step: time.Second})
	return got, reason
}

func toolCalls(steps []*agentrun.Response) []*toolcall.ToolCall {
	var calls []*toolcall.ToolCall
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
	steps, _ := parseLines(t, getTestWords(), startEdit, delta(`{"path":"Note.md","stood":"a"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`)

	calls := toolCalls(steps)
	last := calls[len(calls)-1]
	if last.Tool != "Edit a note" || last.Kind != "edit" || last.About != "Note.md" || last.Path != "Note.md" {
		t.Fatalf("unexpected call: %+v", last)
	}
}

func TestEveryReportOfOneCallCarriesTheNameTheAgentGaveIt(t *testing.T) {
	steps, _ := parseLines(t, getTestWords(), startEdit, delta(`{"path":"Note.md"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`)

	for _, c := range toolCalls(steps) {
		if c.Call != "toolu_01" {
			t.Errorf("call id %q", c.Call)
		}
	}
}

func TestAnUndeclaredToolIsNamedWithoutItsServerPrefix(t *testing.T) {
	steps, _ := parseLines(t, nil,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c","name":"mcp__numen__other_tool"}}}`)

	calls := toolCalls(steps)
	if len(calls) != 1 || calls[0].Tool != "other_tool" || calls[0].Kind != "" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestACollectionCallIsAboutItsFirstElementAndTheRest(t *testing.T) {
	steps, _ := parseLines(t, getTestWords(),
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c","name":"mcp__numen__note_read","input":{"paths":[{"path":"A.md"},{"path":"B.md"},{"path":"C.md"}]}}]}}`)

	calls := toolCalls(steps)
	if len(calls) != 1 || calls[0].About != "A.md and 2 more" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestACallNamingARunOfTextCarriesItsSpan(t *testing.T) {
	steps, _ := parseLines(t, getTestWords(),
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c","name":"mcp__numen__note_edit","input":{"path":"N.md","start":4,"length":6}}]}}`)

	span := toolCalls(steps)[0].Range
	if span.Start != 4 || span.End != 10 {
		t.Fatalf("span %+v", span)
	}
}

func editProgress(steps []*agentrun.Response) []*agentrun.EditProgress {
	var out []*agentrun.EditProgress
	for _, s := range steps {
		if e := s.GetEditProgress(); e != nil {
			out = append(out, e)
		}
	}
	return out
}

func TestNoChangeIsReportedBeforeItsReplacementBegins(t *testing.T) {
	steps, _ := parseLines(t, getTestWords(), startEdit, delta(`{"path":"Note.md",`), delta(`"stood":"A hedgehog`), delta(` is named"`))

	if got := editProgress(steps); len(got) != 0 {
		t.Fatalf("reported early: %+v", got)
	}
}

func TestAChangeIsReportedAsItsReplacementArrives(t *testing.T) {
	steps, _ := parseLines(t, getTestWords(), startEdit,
		delta(`{"path":"Note.md","stood":"A hedgehog",`), delta(`"becomes":"An axe`), delta(`, two-bladed"}`))

	got := editProgress(steps)
	if len(got) != 2 {
		t.Fatalf("expected two reports, got %+v", got)
	}
	if got[0].Path != "Note.md" || got[0].Match != "A hedgehog" || got[0].Text != "An axe" || got[0].Call != "toolu_01" {
		t.Errorf("first report %+v", got[0])
	}
	if got[1].Text != "An axe, two-bladed" {
		t.Errorf("second report %+v", got[1])
	}
}

func TestChangesCloserThanTheIntervalAreReportedOnce(t *testing.T) {
	var got []*agentrun.Response
	p := &streamParser{
		send:  func(r *agentrun.Response) error { got = append(got, r); return nil },
		words: getTestWords(),
		clock: clock.NewFake(time.Unix(100, 0)),
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
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"numen","status":"failed"}]}`:                                       "the agent could not reach this vault",
		`{"type":"system","subtype":"init","mcp_servers":[]}`:                                                                         "the agent was started without this vault",
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"other","status":"failed"},{"name":"numen","status":"connected"}]}`: "",
		`{"type":"system","subtype":"init"}`:                                                                                          "",
		`{"type":"system","subtype":"init","mcp_server_errors":[{"name":"numen","message":"rejected"}]}`:                              "the agent was not given this vault: rejected",
	}
	for line, want := range cases {
		if _, reason := parseLines(t, nil, line); reason != want {
			t.Errorf("%s: got %q want %q", line, reason, want)
		}
	}
}

func TestParseLines_NoVaultExpected(t *testing.T) {
	line := `{"type":"system","subtype":"init","mcp_servers":[]}`
	reason := Parse(t.Context(), strings.NewReader(line+"\n"), func(*agentrun.Response) error { return nil }, Options{Clock: &steppingClock{tick: time.Unix(0, 0), step: time.Second}})
	if reason != "" {
		t.Fatalf("expected empty reason when no vault is expected, got %q", reason)
	}
}

func TestParseLines_ErrorResultFormatting(t *testing.T) {
	cases := map[string]string{
		`{"type":"result","is_error":true,"subtype":"error_max_turns"}`:                            "the agent reached the limit of steps without finishing",
		`{"type":"result","is_error":true,"subtype":"error_context_length_exceeded"}`:              "the conversation is too long for the model",
		`{"type":"result","is_error":true,"subtype":"error_overloaded"}`:                           "the model is overloaded",
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

func TestAChangeIsLocatedByThePathArgumentAlone(t *testing.T) {
	var got []*agentrun.Response
	p := &streamParser{
		send:  func(r *agentrun.Response) error { got = append(got, r); return nil },
		words: map[string]toolcall.Declaration{"note_edit": {About: "title", Match: "stood", Text: "becomes"}},
		clock: clock.NewFake(time.Unix(100, 0)),
	}
	p.tool, p.callID = "mcp__numen__note_edit", "c"
	p.arguments.WriteString(`{"title":"T","stood":"x","becomes":"y`)
	p.reportEdit()

	if len(editProgress(got)) != 0 {
		t.Fatalf("a title was reported as a path: %+v", editProgress(got))
	}
}

func TestStreamParserPreviewIntervalPacing(t *testing.T) {
	var got []*agentrun.Response
	send := func(r *agentrun.Response) error {
		got = append(got, r)
		return nil
	}
	words := map[string]toolcall.Declaration{
		"note_edit": {About: "Edit a note", Match: "path", Text: "text"},
	}
	r := strings.NewReader(strings.Join([]string{
		startEdit,
		delta(`{"path":"a.md","text":"first"}`),
		delta(`{"path":"a.md","text":"second"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n") + "\n")

	parseStreamWithClock(t.Context(), r, send, words, nil, &steppingClock{tick: time.Unix(0, 0), step: 50 * time.Millisecond})

	edits := editProgress(got)
	if len(edits) != 2 {
		t.Fatalf("expected 2 edit progress steps spaced by 50ms, got %d", len(edits))
	}

	// At 49ms pacing, the second edit within the interval is suppressed.
	got = nil
	r49 := strings.NewReader(strings.Join([]string{
		startEdit,
		delta(`{"path":"a.md","text":"first"}`),
		delta(`{"path":"a.md","text":"second"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n") + "\n")
	parseStreamWithClock(t.Context(), r49, send, words, nil, &steppingClock{tick: time.Unix(0, 0), step: 49 * time.Millisecond})
	if len(editProgress(got)) != 1 {
		t.Fatalf("expected 1 edit progress step when spaced by 49ms, got %d", len(editProgress(got)))
	}
}

func TestStreamParserInitSessionRecord(t *testing.T) {
	send := func(_ *agentrun.Response) error {
		return nil
	}
	var recorded []string
	record := func(sess string) { recorded = append(recorded, sess) }

	r := strings.NewReader(`{"type":"system","subtype":"init","session_id":"sess-123"}` + "\n")
	parseStreamWithClock(t.Context(), r, send, nil, record, &steppingClock{tick: time.Unix(0, 0), step: time.Second})
	if len(recorded) != 1 || recorded[0] != "sess-123" {
		t.Fatalf("expected one recorded session %q, got %v", "sess-123", recorded)
	}

	r2 := strings.NewReader(`{"type":"system","subtype":"init","session_id":""}` + "\n")
	parseStreamWithClock(t.Context(), r2, send, nil, record, &steppingClock{tick: time.Unix(0, 0), step: time.Second})
	if len(recorded) != 1 {
		t.Fatalf("expected empty session to not be recorded, got %v", recorded)
	}
}

func TestStreamParserToolCallChunkAndReset(t *testing.T) {
	var got []*agentrun.Response
	send := func(r *agentrun.Response) error {
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

	parseStreamWithClock(t.Context(), r, send, nil, nil, &steppingClock{tick: time.Unix(0, 0), step: time.Second})

	calls := toolCalls(got)
	var t2Calls []*toolcall.ToolCall
	for _, c := range calls {
		if c.Tool == "t2" {
			t2Calls = append(t2Calls, c)
		}
	}
	if len(t2Calls) != 3 {
		t.Fatalf("expected exactly 3 calls for t2, got %d (%+v)", len(t2Calls), t2Calls)
	}
	if t2Calls[1].Length != 200 {
		t.Errorf("expected written 200 for chunk, got %d", t2Calls[1].Length)
	}
	var t3Calls []*toolcall.ToolCall
	for _, c := range calls {
		if c.Tool == "t3" {
			t3Calls = append(t3Calls, c)
		}
	}
	if len(t3Calls) == 0 || t3Calls[0].Call != "" {
		t.Fatalf("expected empty call id for t3, got %+v", t3Calls)
	}
}

func TestStreamParserAssistantMessageIgnoredWhenStreaming(t *testing.T) {
	var got []*agentrun.Response
	send := func(r *agentrun.Response) error {
		got = append(got, r)
		return nil
	}
	r := strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`,
	}, "\n") + "\n")

	parseStreamWithClock(t.Context(), r, send, nil, nil, &steppingClock{tick: time.Unix(0, 0), step: time.Second})

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
	var got []*agentrun.Response
	send := func(r *agentrun.Response) error {
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

	parseStreamWithClock(t.Context(), r, send, words, nil, &steppingClock{tick: time.Unix(0, 0), step: time.Second})

	for _, s := range got {
		if ep := s.GetEditProgress(); ep != nil {
			t.Fatalf("expected no edit progress when tool text is empty, got %+v", ep)
		}
	}
}

func TestStreamParserEdgeCasesAndBoundaries(t *testing.T) {
	// Canceled context returns "canceled" immediately.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if got := Parse(ctx, strings.NewReader(""), func(*agentrun.Response) error { return nil }, Options{}); got != "canceled" {
		t.Errorf("parseStream canceled = %q, want \"canceled\"", got)
	}

	// Send error stops subsequent sends and becomes the reason unless result error was set.
	sendCalls := 0
	reason := Parse(t.Context(), strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"one"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"two"}}}`,
	}, "\n")+"\n"), func(*agentrun.Response) error {
		sendCalls++
		return os.ErrClosed
	}, Options{})
	if sendCalls != 1 || reason != os.ErrClosed.Error() {
		t.Errorf("send error: calls=%d reason=%q", sendCalls, reason)
	}

	reasonWithResult := Parse(t.Context(), strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"one"}}}`,
		`{"type":"result","is_error":true,"subtype":"error_rate_limit"}`,
	}, "\n")+"\n"), func(*agentrun.Response) error {
		return os.ErrClosed
	}, Options{})
	if reasonWithResult != "the rate limit was reached" {
		t.Errorf("reasonWithResult = %q, want rate limit reason", reasonWithResult)
	}

	// Partial invalid JSON in parseLine, hasToolAnswer, and parseAssistantMessage does not emit steps.
	if (event{Message: []byte(`{"content":[{"type":"tool_result"}],bad}`)}).hasToolAnswer() {
		t.Error("expected hasToolAnswer false on invalid JSON")
	}
	steps, badReason := parseLines(t, getTestWords(),
		`{"type":"result","is_error":true,bad`,
		`{"type":"system","subtype":"init","session_id":"s-no-record"}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"bad"}],bad}`,
	)
	if len(steps) != 0 || badReason != "" {
		t.Errorf("expected no steps and empty reason on ignored events, got %d steps, reason %q", len(steps), badReason)
	}

	// content_block_start (tool_use) sets isStreaming=true (ignoring later assistant message),
	// resets offset=0 and last=time.Time{}, and content_block_stop clears tool state.
	var collected []*agentrun.Response
	fakeClock := clock.NewFake(time.Unix(100, 0))
	chunk199 := strings.Repeat("a", 199)
	parseStreamWithClock(t.Context(), strings.NewReader(strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c1","name":"mcp__numen__note_edit"}}}`,
		delta(`{"path":"a.md","stood":"m","becomes":"` + chunk199 + `"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		delta(chunk199),
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c2","name":"mcp__numen__note_edit"}}}`,
		delta(chunk199),
		delta("b"),
		delta(`{"path":"b.md","stood":"m2","becomes":"t2"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"ignored"}]}}`,
	}, "\n")+"\n"), func(r *agentrun.Response) error {
		collected = append(collected, r)
		return nil
	}, getTestWords(), nil, fakeClock)

	// Both c1 and c2 must emit EditProgress even though fakeClock did not advance between c1 and c2.
	if gotEdits := editProgress(collected); len(gotEdits) != 2 {
		t.Errorf("expected 2 edit progress events across two tool blocks with fixed clock, got %d", len(gotEdits))
	}
	// c2 had 199 chars (no chunk emit), then +1 char (=200, 1 chunk emit), then stop (1 stop emit), plus start (1 start emit) = 3 calls for c2.
	c2Calls := 0
	for _, c := range toolCalls(collected) {
		if c.Call == "c2" {
			c2Calls++
		}
		if c.Call == "c1" && c.Tool == "" {
			t.Errorf("unexpected empty tool on c1: %+v", c)
		}
	}
	if c2Calls != 3 {
		t.Errorf("expected 3 tool call steps for c2 (start, 200-char chunk, stop), got %d", c2Calls)
	}
	for _, s := range collected {
		if s.GetText() != "" {
			t.Errorf("expected assistant message to be ignored after tool_use streaming, got %q", s.GetText())
		}
	}

	// reportEdit when words.Text is empty (even when args contain "") or when match is empty does not emit.
	var noEdits []*agentrun.Response
	parseStreamWithClock(t.Context(), strings.NewReader(strings.Join([]string{
		startEdit,
		delta(`{"path":"a.md","stood":"","becomes":"val"}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c2","name":"mcp__numen__note_read"}}}`,
		delta(`{"path":"a.md","empty":""}`),
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
	}, "\n")+"\n"), func(r *agentrun.Response) error {
		noEdits = append(noEdits, r)
		return nil
	}, getTestWords(), nil, &steppingClock{tick: time.Unix(0, 0), step: time.Second})
	if len(editProgress(noEdits)) != 0 {
		t.Errorf("expected 0 edit progress steps when match or words.Text is empty, got %d", len(editProgress(noEdits)))
	}
}
