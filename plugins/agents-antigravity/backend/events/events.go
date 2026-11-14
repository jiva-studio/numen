// Package events reads the JSON events the Antigravity program prints and hands what they say to a sink.
package events

import (
	"encoding/json"
	"strings"
)

// Sink receives what the events of a run say. A call is reported once its arguments are whole; the
// program reports no end of a call. Stop receives the reason the program gave for a run that did not
// finish.
type Sink interface {
	SendText(text string) error
	SendToolCall(tool, arguments string) error
	SetSession(session string)
	Stop(reason string)
	Finish()
}

type eventReader struct {
	sink    Sink
	sendErr error

	isTextStreamed bool

	currentTool string
	currentArgs strings.Builder
}

// Handler turns the decoded events of the program's output into reports for a sink.
type Handler struct {
	reader *eventReader
}

// New returns the handler that hands what each event says to sink.
func New(sink Sink) *Handler {
	return &Handler{reader: &eventReader{sink: sink, sendErr: nil, isTextStreamed: false, currentTool: "", currentArgs: strings.Builder{}}}
}

// Handle hands what one event says to the sink. Once the sink refused a report, it is handed no other.
func (h *Handler) Handle(event map[string]any) {
	h.reader.handle(event)
}

// SendText hands a line of the output that is not an event to the sink as text.
func (h *Handler) SendText(text string) {
	h.reader.sendText(text)
}

// Err returns the error of the first report the sink refused.
func (h *Handler) Err() error {
	return h.reader.sendErr
}

func (r *eventReader) sendText(text string) {
	if r.sendErr == nil {
		r.sendErr = r.sink.SendText(text)
	}
}

func (r *eventReader) sendToolCall(tool, arguments string) {
	if r.sendErr == nil {
		r.sendErr = r.sink.SendToolCall(tool, arguments)
	}
}

func (r *eventReader) handle(rawMap map[string]any) {
	if kind := getString(rawMap, "event"); kind != "" {
		r.handleCliEvent(kind, rawMap)
		return
	}
	if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
		r.handleStreamEvent(rawMap)
		return
	}

	for _, key := range []string{"conversation_id", "session_id"} {
		if session, ok := rawMap[key].(string); ok {
			r.sink.SetSession(session)
		}
	}
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.sendText(content)
	}
	r.handleStepUpdate(rawMap)
	r.handleResult(rawMap)
}

func (r *eventReader) handleStreamEvent(rawMap map[string]any) {
	evObj, ok := rawMap["event"].(map[string]any)
	if !ok {
		return
	}
	switch getString(evObj, "type") {
	case "content_block_start":
		r.startBlock(evObj)
	case "content_block_delta":
		r.continueBlock(evObj)
	case "content_block_stop":
		if r.currentTool != "" {
			r.sendToolCall(r.currentTool, r.currentArgs.String())
			r.currentTool = ""
			r.currentArgs.Reset()
		}
	}
}

func (r *eventReader) startBlock(evObj map[string]any) {
	cb, ok := evObj["content_block"].(map[string]any)
	if !ok {
		return
	}
	if cb["type"] != "tool_use" {
		return
	}
	r.currentTool = getString(cb, "name")
	r.currentArgs.Reset()
	initInput, ok := cb["input"].(map[string]any)
	if !ok {
		return
	}
	if raw, err := json.Marshal(initInput); err == nil {
		r.currentArgs.Write(raw)
	}
}

func (r *eventReader) continueBlock(evObj map[string]any) {
	delta, ok := evObj["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.sendText(text)
	}
	if partial, ok := delta["partial_json"].(string); ok {
		r.currentArgs.WriteString(partial)
	}
}

func (r *eventReader) handleToolCalls(rawMap map[string]any) {
	toolCalls, ok := rawMap["tool_calls"].([]any)
	if !ok {
		return
	}
	for _, tcRaw := range toolCalls {
		tc, ok := tcRaw.(map[string]any)
		if !ok {
			continue
		}
		args, err := json.Marshal(tc["arguments"])
		if err != nil {
			args = nil
		}
		r.sendToolCall(getString(tc, "name"), string(args))
	}
}

func (r *eventReader) handleStepUpdate(rawMap map[string]any) {
	delta, ok := rawMap["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.sendText(text)
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.sendText(text)
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.sink.Stop(reason)
	}
}

func (r *eventReader) handleCliEvent(kind string, rawMap map[string]any) {
	switch kind {
	case "step_update":
		r.handleCliStep(getMap(rawMap, "step_update"))
	case "result":
		r.handleCliResult(getMap(rawMap, "result"))
	default:
		if session, ok := rawMap["conversation_id"].(string); ok {
			r.sink.SetSession(session)
		}
	}
}

func (r *eventReader) handleCliStep(step map[string]any) {
	if session, ok := step["conversation_id"].(string); ok {
		r.sink.SetSession(session)
	}
	text := getString(step, "text_delta")
	if kind := getString(step, "step_type"); kind != "agent_response" || text == "" {
		return
	}
	r.isTextStreamed = true
	r.sendText(text)
}

func (r *eventReader) handleCliResult(result map[string]any) {
	r.sink.Finish()
	if session, ok := result["conversation_id"].(string); ok {
		r.sink.SetSession(session)
	}
	if status := getString(result, "status"); status != "" && !strings.EqualFold(status, "success") {
		reason := getString(result, "error")
		if reason == "" {
			reason = "the run ended with status " + status
		}
		r.sink.Stop(reason)
		return
	}
	if response := getString(result, "response"); response != "" && !r.isTextStreamed {
		r.sendText(response)
	}
}
