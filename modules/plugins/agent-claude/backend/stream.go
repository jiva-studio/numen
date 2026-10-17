package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// serverName is what the vault's tool server calls itself, and the prefix its tools arrive under.
const serverName = "numen"

// reportChunk is how many characters of a call are written between two reports of it.
const reportChunk = 200

// previewInterval is the shortest time between two reports of a change being written.
const previewInterval = 50 * time.Millisecond

// toolWords is how one vault tool is spoken about and read while a call of it is written.
type toolWords struct {
	title, kind, about, element, match, text string
}

type streamParser struct {
	send        func(*numenv1.AskAgentResponse) error
	words       map[string]toolWords
	now         func() time.Time
	last        time.Time
	record      func(string)
	isStreaming bool
	callID      string
	tool        string
	arguments   strings.Builder
	offset      int
	reason      string
}

type event struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	Session string          `json:"session_id"`
	Message json.RawMessage `json:"message"`
	Event   streamEvent     `json:"event"`
	Result  string          `json:"result"`
	IsError bool            `json:"is_error"`
	Status  string          `json:"status"`

	Servers *[]struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers"`
	ServerErrors []struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"mcp_server_errors"`
}

func (e event) hasToolAnswer() bool {
	if len(e.Message) == 0 {
		return false
	}
	var msg struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(e.Message, &msg); err != nil {
		return false
	}
	for _, c := range msg.Content {
		if c.Type == "tool_result" {
			return true
		}
	}
	return false
}

type streamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Partial string `json:"partial_json"`
	} `json:"delta"`
	Block struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		ID    string          `json:"id"`
		Input json.RawMessage `json:"input"`
	} `json:"content_block"`
}

func parseStream(
	ctx context.Context,
	r io.Reader,
	send func(*numenv1.AskAgentResponse) error,
	words map[string]toolWords,
	record func(string),
) string {
	return parseStreamWithClock(ctx, r, send, words, record, time.Now)
}

// parseStreamWithClock is parseStream with the clock that paces the reports of a change.
func parseStreamWithClock(
	ctx context.Context,
	r io.Reader,
	send func(*numenv1.AskAgentResponse) error,
	words map[string]toolWords,
	record func(string),
	now func() time.Time,
) string {
	p := &streamParser{
		send:   send,
		words:  words,
		now:    now,
		record: record,
	}
	scanner := bufio.NewReader(r)
	for {
		if err := ctx.Err(); err != nil {
			return "canceled"
		}
		line, err := scanner.ReadString('\n')
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			p.parseLine(ctx, trimmed)
		}
		if err != nil {
			return p.reason
		}
	}
}

func (p *streamParser) parseLine(ctx context.Context, line string) {
	var said event
	if err := json.Unmarshal([]byte(line), &said); err != nil {
		return
	}

	switch said.Type {
	case "stream_event":
		p.parseStreamEvent(ctx, said.Event)
	case "assistant":
		p.parseAssistantMessage(ctx, said)
	case "user":
		if said.hasToolAnswer() {
			_ = p.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ToolFinished{
					ToolFinished: &numenv1.ToolFinished{},
				},
			})
		}
	case "system":
		switch said.Subtype {
		case "init":
			if said.Session != "" && p.record != nil {
				p.record(said.Session)
			}
			p.setReason(getUnreachableReason(said))
		case "status":
			if said.Status == "requesting" {
				_ = p.send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_ModelStarted{
						ModelStarted: &numenv1.ModelStarted{},
					},
				})
			}
		}
	case "result":
		if said.IsError {
			p.setReason(getResultError(said))
			return
		}
		p.setReason("")
	}
}

func (p *streamParser) parseStreamEvent(_ context.Context, ev streamEvent) {
	switch ev.Type {
	case "content_block_start":
		if ev.Block.Type == "tool_use" {
			p.isStreaming = true
			p.callID = ev.Block.ID
			p.tool = ev.Block.Name
			p.arguments.Reset()
			p.offset = 0
			p.last = time.Time{}
			_ = p.send(p.buildCallStep(p.callID, p.tool, ""))
		}
	case "content_block_delta":
		switch ev.Delta.Type {
		case "text_delta":
			if ev.Delta.Text == "" {
				return
			}
			p.isStreaming = true
			_ = p.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Said{
					Said: ev.Delta.Text,
				},
			})
		case "input_json_delta":
			p.arguments.WriteString(ev.Delta.Partial)
			if p.arguments.Len()-p.offset >= reportChunk {
				p.offset = p.arguments.Len()
				_ = p.send(p.buildCallStep(p.callID, p.tool, p.arguments.String()))
			}
			p.reportEdit()
		}
	case "content_block_stop":
		if p.tool == "" {
			return
		}
		_ = p.send(p.buildCallStep(p.callID, p.tool, p.arguments.String()))
		p.callID = ""
		p.tool = ""
		p.arguments.Reset()
	}
}

func (p *streamParser) parseAssistantMessage(_ context.Context, said event) {
	if p.isStreaming || len(said.Message) == 0 {
		return
	}
	var msg struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(said.Message, &msg); err != nil {
		return
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				_ = p.send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_Said{
						Said: block.Text,
					},
				})
			}
		case "tool_use":
			_ = p.send(p.buildCallStep(block.ID, block.Name, string(block.Input)))
		}
	}
}

func (p *streamParser) setReason(why string) {
	if p.reason == "" {
		p.reason = why
	}
}

func stripServerPrefix(tool string) string {
	if name, ok := strings.CutPrefix(tool, "mcp__"+serverName+"__"); ok {
		return name
	}
	if strings.HasPrefix(tool, "mcp__") {
		if parts := strings.SplitN(tool, "__", 3); len(parts) == 3 {
			return parts[2]
		}
	}
	return tool
}

// buildCallStep is a tool call as the person is told about it. A tool the vault declared is shown
// by its title, with its effect and what the call is about read by the arguments it declared.
func (p *streamParser) buildCallStep(callID, tool, args string) *numenv1.AskAgentResponse {
	name := stripServerPrefix(tool)
	words, declared := p.words[name]
	call := &numenv1.ToolCall{Tool: name, Call: callID, Written: int32(len([]rune(args)))}
	if declared {
		if words.title != "" {
			call.Tool = words.title
		}
		call.Kind = words.kind
		if words.about == "" {
			call.Written = 0
		} else {
			call.About = getAbout(words, args)
			if words.about == "path" {
				call.Path, call.Span = call.About, getSpan(args)
			}
		}
	} else {
		call.About, call.Path, call.Span = extractCallDetails(args)
	}
	return &numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_ToolCall{ToolCall: call}}
}

// getSpan is the run of a source's text a call names, read once the arguments parse whole.
func getSpan(args string) *numenv1.Span {
	var made map[string]any
	if err := json.Unmarshal([]byte(args), &made); err != nil {
		return nil
	}
	start, _ := made["start"].(float64)
	length, _ := made["length"].(float64)
	if length <= 0 {
		return nil
	}
	return &numenv1.Span{From: int32(start), To: int32(start + length)}
}

// getAbout is what a call is about, read from the arguments as far as they have arrived.
func getAbout(words toolWords, args string) string {
	var made map[string]any
	if err := json.Unmarshal([]byte(args), &made); err != nil {
		if seen := getPartialJSONField(args, words.element); seen != "" {
			return seen
		}
		return getPartialJSONField(args, words.about)
	}
	switch value := made[words.about].(type) {
	case string:
		return value
	case []any:
		return describeCollection(value, words.element)
	}
	return ""
}

// describeCollection is the first element of a collection by the name it carries, and how many
// others there are.
func describeCollection(value []any, inside string) string {
	if len(value) == 0 {
		return ""
	}
	first := ""
	switch element := value[0].(type) {
	case string:
		first = element
	case map[string]any:
		if inside == "" {
			return ""
		}
		first, _ = element[inside].(string)
	}
	if first == "" {
		return ""
	}
	if len(value) > 1 {
		return fmt.Sprintf("%s and %d more", first, len(value)-1)
	}
	return first
}

// reportEdit reports the change a call is writing, once the text that replaces the match has
// begun to arrive. A change is reported at most once every previewInterval.
func (p *streamParser) reportEdit() {
	words, declared := p.words[stripServerPrefix(p.tool)]
	if !declared || words.text == "" {
		return
	}
	args := p.arguments.String()
	if !strings.Contains(args, `"`+words.text+`"`) {
		return
	}
	path, match := getPartialJSONField(args, "path"), getPartialJSONField(args, words.match)
	if path == "" || match == "" {
		return
	}
	now := p.now()
	if now.Sub(p.last) < previewInterval {
		return
	}
	p.last = now
	_ = p.send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_EditProgress{
		EditProgress: &numenv1.EditProgress{
			Call:  p.callID,
			Path:  path,
			Match: match,
			Text:  getPartialJSONField(args, words.text),
		},
	}})
}

// getUnreachableReason is why the vault's tools did not arrive, empty when they did and when the
// line says nothing about servers.
func getUnreachableReason(said event) string {
	for _, skipped := range said.ServerErrors {
		if skipped.Name == serverName {
			return fmt.Sprintf("the agent was not given this vault: %s", skipped.Message)
		}
	}
	if said.Servers == nil {
		return ""
	}
	for _, server := range *said.Servers {
		if server.Name != serverName {
			continue
		}
		if server.Status == "failed" {
			return "the agent could not reach this vault"
		}
		return ""
	}
	return "the agent was started without this vault"
}

// getResultError is what the last line says went wrong.
func getResultError(said event) string {
	if text := strings.TrimSpace(said.Result); text != "" && text != said.Subtype {
		return text
	}
	if said.Subtype != "" {
		return formatSubtype(said.Subtype)
	}
	return "the agent stopped without finishing"
}

// formatSubtype words a machine error subtype for a person.
func formatSubtype(subtype string) string {
	switch subtype {
	case "error_max_turns", "max_turns":
		return "the agent reached the limit of steps without finishing"
	case "error_context_length_exceeded", "error_context_length", "context_length_exceeded":
		return "the conversation is too long for the model"
	case "error_rate_limit", "rate_limit":
		return "the rate limit was reached"
	case "error_authentication", "error_auth", "authentication_error":
		return "authentication with the model failed"
	case "error_overloaded", "overloaded":
		return "the model is overloaded"
	case "error_interrupted", "error_cancelled", "interrupted":
		return "the agent was interrupted"
	default:
		if rest, ok := strings.CutPrefix(subtype, "error_"); ok {
			return strings.ReplaceAll(rest, "_", " ")
		}
		return subtype
	}
}

func extractCallDetails(args string) (string, string, *numenv1.Span) {
	if args == "" {
		return "", "", nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err == nil {
		var about, path string
		if p, ok := parsed["path"].(string); ok {
			path = p
		} else if f, ok := parsed["file"].(string); ok {
			path = f
		}

		if q, ok := parsed["query"].(string); ok {
			about = q
		} else if a, ok := parsed["about"].(string); ok {
			about = a
		} else if t, ok := parsed["title"].(string); ok {
			about = t
		} else if path != "" {
			about = path
		} else {
			about = args
		}

		var span *numenv1.Span
		startVal, hasStart := parsed["start"]
		lenVal, hasLen := parsed["length"]
		if hasStart && hasLen {
			var from, length float64
			if s, ok := startVal.(float64); ok {
				from = s
			}
			if l, ok := lenVal.(float64); ok {
				length = l
			}
			if from >= 0 && length >= 0 && from <= math.MaxInt32-length {
				span = &numenv1.Span{
					From: int32(from),
					To:   int32(from + length),
				}
			}
		}
		return about, path, span
	}

	// Partial JSON fallback
	path := getPartialJSONField(args, "path")
	if path == "" {
		path = getPartialJSONField(args, "file")
	}
	query := getPartialJSONField(args, "query")
	about := query
	if about == "" {
		about = getPartialJSONField(args, "about")
	}
	if about == "" {
		about = getPartialJSONField(args, "title")
	}
	if about == "" {
		about = path
	}

	var span *numenv1.Span
	startStr := getPartialJSONField(args, "start")
	lenStr := getPartialJSONField(args, "length")
	if startStr != "" && lenStr != "" {
		if s, err := strconv.ParseInt(startStr, 10, 32); err == nil {
			if l, err := strconv.ParseInt(lenStr, 10, 32); err == nil {
				if s >= 0 && l >= 0 && s <= math.MaxInt32-l {
					span = &numenv1.Span{
						From: int32(s),
						To:   int32(s + l),
					}
				}
			}
		}
	}

	return about, path, span
}
