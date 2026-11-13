package codexevent

import (
	"errors"
	"slices"
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

func (r *recorder) RecordThread(id string) {
	r.reports = append(r.reports, "thread "+id)
}

func (r *recorder) SendThought(text string) error {
	r.reports = append(r.reports, "thought "+text)
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

func handleAll(t *testing.T, sink *recorder, events ...map[string]any) {
	t.Helper()
	for _, one := range events {
		if err := Handle(sink, one); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnAgentMessageBecomesText(t *testing.T) {
	var sink recorder

	handleAll(t, &sink,
		map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": "The answer."}},
		map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": ""}},
	)

	if !slices.Equal(sink.reports, []string{"text The answer."}) {
		t.Errorf("reports %q", sink.reports)
	}
}

func TestAToolCallStartsAndFinishes(t *testing.T) {
	var sink recorder

	handleAll(t, &sink,
		map[string]any{"type": "item.started", "item": map[string]any{
			"id": "i1", "type": "mcp_tool_call", "name": "read_note", "arguments": map[string]any{"path": "a.md"},
		}},
		map[string]any{"type": "item.completed", "item": map[string]any{"id": "i1", "type": "mcp_tool_call", "status": "completed"}},
	)

	if !slices.Equal(sink.reports, []string{`call i1 read_note {"path":"a.md"}`, "result i1 done"}) {
		t.Errorf("reports %q", sink.reports)
	}
}

func TestAToolCallThatFailedIsReportedSo(t *testing.T) {
	var sink recorder

	handleAll(t, &sink,
		map[string]any{"type": "item.completed", "item": map[string]any{"id": "i1", "type": "mcp_tool_call", "status": "failed"}},
		map[string]any{"type": "item.completed", "item": map[string]any{"id": "i2", "type": "mcp_tool_call", "error": map[string]any{"message": "no"}}},
	)

	if !slices.Equal(sink.reports, []string{"result i1 failed", "result i2 failed"}) {
		t.Errorf("reports %q", sink.reports)
	}
}

func TestAToolCallNamesItsToolByTheToolField(t *testing.T) {
	var sink recorder

	handleAll(t, &sink, map[string]any{"type": "item.started", "item": map[string]any{
		"id": "i1", "type": "mcp_tool_call", "server": "numen", "tool": "search_notes", "arguments": map[string]any{"q": "x"},
	}})

	if !slices.Equal(sink.reports, []string{`call i1 search_notes {"q":"x"}`}) {
		t.Errorf("reports %q", sink.reports)
	}
}

func TestOtherEventsSendNothing(t *testing.T) {
	var sink recorder

	handleAll(t, &sink,
		map[string]any{"type": "item.completed"},
		map[string]any{"type": "item.started", "item": map[string]any{"type": "agent_message", "text": "no"}},
		map[string]any{"type": "other", "item": map[string]any{"type": "mcp_tool_call", "tool": "t"}},
		map[string]any{"type": "item.started", "item": map[string]any{"type": "other"}},
		map[string]any{"type": "item.completed", "item": map[string]any{"type": "other"}},
		map[string]any{"type": "item.started", "item": map[string]any{"type": "reasoning", "text": "early"}},
		map[string]any{"type": "item.completed", "item": map[string]any{"type": "reasoning", "text": ""}},
		map[string]any{"type": "thread.started"},
	)

	if len(sink.reports) != 0 {
		t.Errorf("reports %q", sink.reports)
	}
}

func TestTheThreadAndTheReasoningOfTheRunAreReported(t *testing.T) {
	var sink recorder

	handleAll(t, &sink,
		map[string]any{"type": "thread.started", "thread_id": "0199a213-81c0-7800-8aa1-bbab2a035a53"},
		map[string]any{"type": "item.completed", "item": map[string]any{"type": "reasoning", "text": "The beds face south."}},
	)

	want := []string{"thread 0199a213-81c0-7800-8aa1-bbab2a035a53", "thought The beds face south."}
	if !slices.Equal(sink.reports, want) {
		t.Errorf("reports %q, want %q", sink.reports, want)
	}
}

func TestTheErrorOfTheSinkIsReturned(t *testing.T) {
	sink := recorder{refusal: errors.New("closed")}

	err := Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": "x"}})

	if !errors.Is(err, sink.refusal) {
		t.Errorf("Handle = %v, want the error of the sink", err)
	}
}

func TestFieldReadersIgnoreValuesOfAnotherType(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}
	if got := getToolArguments(map[string]any{"arguments": make(chan int)}); got != "" {
		t.Errorf("getToolArguments channel = %q, want empty", got)
	}
}
