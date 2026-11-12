package stream

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// recorder is a sink that writes down each report as one line.
type recorder struct {
	reports []string
	refusal error
}

func (r *recorder) SendText(text string) error {
	r.reports = append(r.reports, "text "+text)
	return r.refusal
}

func (r *recorder) SendToolCall(id, tool, arguments string) error {
	r.reports = append(r.reports, "call "+id+" "+tool+" "+arguments)
	return r.refusal
}

func (r *recorder) SendToolResult(id string, isFailed bool) error {
	state := "done"
	if isFailed {
		state = "failed"
	}
	r.reports = append(r.reports, "result "+id+" "+state)
	return r.refusal
}

func parseLines(t *testing.T, lines ...string) ([]string, string) {
	t.Helper()
	sink := &recorder{}
	reason, err := Parse(t.Context(), strings.NewReader(strings.Join(lines, "\n")+"\n"), sink, Options{ExpectsVault: true})
	if err != nil {
		t.Fatal(err)
	}
	return sink.reports, reason
}

const startEdit = `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"toolu_01","name":"mcp__numen__note_edit"}}}`

const stopBlock = `{"type":"stream_event","event":{"type":"content_block_stop"}}`

func delta(partial string) string {
	return `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":` +
		quote(partial) + `}}}`
}

func quote(text string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text) + `"`
}

func textDelta(text string) string {
	return `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":` + quote(text) + `}}}`
}

// A call is reported when it starts and once more with its whole arguments, by the name of the vault's
// tool without the prefix of its server.
func TestACallIsReportedAtItsStartAndWithItsWholeArguments(t *testing.T) {
	reports, _ := parseLines(t, startEdit, delta(`{"path":"Beds.md",`), delta(`"text":"Raised"}`), stopBlock)

	want := []string{
		"call toolu_01 note_edit ",
		`call toolu_01 note_edit {"path":"Beds.md","text":"Raised"}`,
	}
	if !slices.Equal(reports, want) {
		t.Errorf("reports = %q, want %q", reports, want)
	}
}

func TestAToolOfAnotherServerIsNamedWithoutItsPrefix(t *testing.T) {
	reports, _ := parseLines(t,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"c","name":"mcp__weather__forecast"}}}`)

	if !slices.Equal(reports, []string{"call c forecast "}) {
		t.Errorf("reports = %q", reports)
	}
}

func TestTheEndOfEachCallIsReportedWithWhetherItFailed(t *testing.T) {
	reports, _ := parseLines(t,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"c1","content":"ok"},{"type":"tool_result","tool_use_id":"c2","is_error":true}]}}`)

	if !slices.Equal(reports, []string{"result c1 done", "result c2 failed"}) {
		t.Errorf("reports = %q", reports)
	}
}

func TestAWholeMessageIsReportedWhenTheProgramStreamsNothing(t *testing.T) {
	reports, _ := parseLines(t,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Raised beds."},{"type":"tool_use","id":"c","name":"mcp__numen__note_read","input":{"path":"Beds.md"}}]}}`)

	want := []string{"text Raised beds.", `call c note_read {"path":"Beds.md"}`}
	if !slices.Equal(reports, want) {
		t.Errorf("reports = %q, want %q", reports, want)
	}
}

func TestTheWholeMessageOfAStreamedAnswerIsNotReportedAgain(t *testing.T) {
	reports, _ := parseLines(t, textDelta("hello"), textDelta(""),
		`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`)

	if !slices.Equal(reports, []string{"text hello"}) {
		t.Errorf("reports = %q", reports)
	}
}

func TestTheVaultThatTheProgramCannotReachIsTheReasonOfTheRun(t *testing.T) {
	cases := map[string]string{
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"numen","status":"failed"}]}`:                                       "the agent could not reach this vault",
		`{"type":"system","subtype":"init","mcp_servers":[]}`:                                                                         "the agent was started without this vault",
		`{"type":"system","subtype":"init","mcp_servers":[{"name":"other","status":"failed"},{"name":"numen","status":"connected"}]}`: "",
		`{"type":"system","subtype":"init"}`:                                                                                          "",
		`{"type":"system","subtype":"init","mcp_server_errors":[{"name":"numen","message":"rejected"}]}`:                              "the agent was not given this vault: rejected",
	}
	for line, want := range cases {
		if _, reason := parseLines(t, line); reason != want {
			t.Errorf("%s: got %q want %q", line, reason, want)
		}
	}
}

func TestARunThatExpectsNoVaultHasNoReasonForAMissingOne(t *testing.T) {
	reason, err := Parse(t.Context(), strings.NewReader(`{"type":"system","subtype":"init","mcp_servers":[]}`+"\n"), &recorder{}, Options{})
	if err != nil || reason != "" {
		t.Fatalf("Parse = %q, %v, want no reason", reason, err)
	}
}

func TestTheErrorOfTheResultIsSaidInPlainWords(t *testing.T) {
	cases := map[string]string{
		`{"type":"result","is_error":true,"subtype":"error_max_turns"}`:                            ReasonMaxTurns,
		`{"type":"result","is_error":true,"subtype":"error_context_length_exceeded"}`:              "the conversation is too long for the model",
		`{"type":"result","is_error":true,"subtype":"error_overloaded"}`:                           "the model is overloaded",
		`{"type":"result","is_error":true,"subtype":"error_something_new"}`:                        "something new",
		`{"type":"result","is_error":true,"subtype":"error_max_turns","result":"error_max_turns"}`: ReasonMaxTurns,
		`{"type":"result","is_error":true,"result":"overloaded, try later"}`:                       "overloaded, try later",
		`{"type":"result","is_error":true}`:                                                        "the agent stopped without finishing",
	}
	for line, want := range cases {
		if _, reason := parseLines(t, line); reason != want {
			t.Errorf("%s: got %q want %q", line, reason, want)
		}
	}
}

func TestTheFirstErrorIsTheReason(t *testing.T) {
	_, reason := parseLines(t,
		`{"type":"result","is_error":true,"subtype":"error_rate_limit"}`,
		`{"type":"result","is_error":true,"subtype":"error_overloaded"}`)
	if reason != "the rate limit was reached" {
		t.Fatalf("got %q", reason)
	}
}

func TestTheSessionOfTheProgramIsRecorded(t *testing.T) {
	var recorded []string
	record := func(session string) { recorded = append(recorded, session) }
	lines := `{"type":"system","subtype":"init","session_id":"sess-123"}` + "\n" + `{"type":"system","subtype":"init","session_id":""}` + "\n"

	if _, err := Parse(t.Context(), strings.NewReader(lines), &recorder{}, Options{Record: record}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(recorded, []string{"sess-123"}) {
		t.Fatalf("recorded %v, want the one session", recorded)
	}
}

func TestACanceledRunIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if reason, _ := Parse(ctx, strings.NewReader(textDelta("one")+"\n"), &recorder{}, Options{}); reason != reasonCanceled {
		t.Errorf("reason = %q, want canceled", reason)
	}
}

// A sink that refuses a report is handed no other, and its error ends the parse; the error of the
// result is still the reason.
func TestARefusedReportEndsTheReports(t *testing.T) {
	sink := &recorder{refusal: os.ErrClosed}
	reason, err := Parse(t.Context(), strings.NewReader(strings.Join([]string{
		textDelta("one"), textDelta("two"), `{"type":"result","is_error":true,"subtype":"error_rate_limit"}`,
	}, "\n")+"\n"), sink, Options{})

	if !errors.Is(err, os.ErrClosed) || len(sink.reports) != 1 || reason != "the rate limit was reached" {
		t.Errorf("Parse = %q, %v after %q", reason, err, sink.reports)
	}
}

func TestLinesThatAreNotEventsAreSkipped(t *testing.T) {
	reports, reason := parseLines(t,
		`{"type":"result","is_error":true,bad`,
		`{"type":"user","message":{"content":[{"type":"tool_result"}],bad}}`,
		stopBlock,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"bad"}],bad}`,
	)
	if len(reports) != 0 || reason != "" {
		t.Errorf("got %q and reason %q from lines that are not events", reports, reason)
	}
}
