package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
)

// serverName is what the vault's tool server calls itself, and the prefix its tools arrive under.
const serverName = "numen"

// reportChunk is how many characters of a call are written between two reports of it.
const reportChunk = 200

// previewInterval is the shortest time between two reports of a change being written.
const previewInterval = 50 * time.Millisecond

// timeSource gets the current time.
type timeSource interface {
	Now() time.Time
}

type streamParser struct {
	send         func(*agentrun.Response) error
	words        map[string]toolcall.Declaration
	clock        timeSource
	last         time.Time
	record       func(string)
	isStreaming  bool
	callID       string
	tool         string
	arguments    strings.Builder
	offset       int
	reason       string
	expectsVault bool
	sendErr      error
}

func (p *streamParser) emit(resp *agentrun.Response) {
	if p.sendErr == nil {
		p.sendErr = p.send(resp)
	}
}

func (p *streamParser) getReason() string {
	if p.reason == "" && p.sendErr != nil {
		return p.sendErr.Error()
	}
	return p.reason
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

// streamOptions are what the parser of a stream needs besides the stream and the way to send.
type streamOptions struct {
	words        map[string]toolcall.Declaration
	record       func(string)
	clock        timeSource
	expectsVault bool
}

func parseStreamWithClock(
	ctx context.Context,
	r io.Reader,
	send func(*agentrun.Response) error,
	words map[string]toolcall.Declaration,
	record func(string),
	clock timeSource,
) string {
	return parseStream(ctx, r, send, streamOptions{words: words, record: record, clock: clock, expectsVault: true})
}

func parseStream(ctx context.Context, r io.Reader, send func(*agentrun.Response) error, opts streamOptions) string {
	p := &streamParser{
		send:         send,
		words:        opts.words,
		clock:        opts.clock,
		record:       opts.record,
		expectsVault: opts.expectsVault,
		last:         time.Time{},
		isStreaming:  false,
		callID:       "",
		tool:         "",
		arguments:    strings.Builder{},
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial offset is reset to zero on every content_block_start before tool deltas arrive."
		offset:  0,
		reason:  "",
		sendErr: nil,
	}
	scanner := bufio.NewReader(r)
	for {
		if err := ctx.Err(); err != nil {
			return "canceled"
		}
		line, err := scanner.ReadString('\n')
		if content := strings.TrimSpace(line); content != "" {
			p.parseLine(ctx, content)
		}
		if err != nil {
			return p.getReason()
		}
	}
}

func (p *streamParser) parseLine(ctx context.Context, line string) {
	var response event
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return
	}

	switch response.Type {
	case "stream_event":
		p.parseStreamEvent(ctx, response.Event)
	case "assistant":
		p.parseAssistantMessage(ctx, response)
	case "user":
		if response.hasToolAnswer() {
			p.emit(&agentrun.Response{
				Step: &agentrun.ToolFinishedStep{},
			})
		}
	case "system":
		p.parseSystemEvent(response)
	case "result":
		if response.IsError {
			p.setReason(getResultError(response))
		}
	}
}

func (p *streamParser) parseSystemEvent(response event) {
	switch response.Subtype {
	case "init":
		if response.Session != "" && p.record != nil {
			p.record(response.Session)
		}
		if p.expectsVault {
			p.setReason(getUnreachableReason(response))
		}
	case "status":
		if response.Status == "requesting" {
			p.emit(&agentrun.Response{
				Step: &agentrun.ModelStartedStep{},
			})
		}
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
			p.emit(p.buildCallStep(p.callID, p.tool, ""))
		}
	case "content_block_delta":
		switch ev.Delta.Type {
		case "text_delta":
			if ev.Delta.Text == "" {
				return
			}
			p.isStreaming = true
			p.emit(&agentrun.Response{
				Step: &agentrun.TextStep{
					Text: ev.Delta.Text,
				},
			})
		case "input_json_delta":
			p.arguments.WriteString(ev.Delta.Partial)
			if p.arguments.Len()-p.offset >= reportChunk {
				p.offset = p.arguments.Len()
				p.emit(p.buildCallStep(p.callID, p.tool, p.arguments.String()))
			}
			p.reportEdit()
		}
	case "content_block_stop":
		if p.tool == "" {
			return
		}
		p.emit(p.buildCallStep(p.callID, p.tool, p.arguments.String()))
		p.callID = ""
		p.tool = ""
		p.arguments.Reset()
	}
}

func (p *streamParser) parseAssistantMessage(_ context.Context, response event) {
	if p.isStreaming {
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
	if err := json.Unmarshal(response.Message, &msg); err != nil {
		return
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				p.emit(&agentrun.Response{
					Step: &agentrun.TextStep{
						Text: block.Text,
					},
				})
			}
		case "tool_use":
			p.emit(p.buildCallStep(block.ID, block.Name, string(block.Input)))
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

func (p *streamParser) buildCallStep(callID, tool, args string) *agentrun.Response {
	call := toolcall.Describe(p.words, stripServerPrefix(tool), callID, args)
	return &agentrun.Response{Step: &agentrun.ToolCallStep{ToolCall: call}}
}

func (p *streamParser) reportEdit() {
	words := p.words[stripServerPrefix(p.tool)]
	if words.Text == "" {
		return
	}
	args := p.arguments.String()
	if !strings.Contains(args, `"`+words.Text+`"`) {
		return
	}
	path, match := toolcall.ReadPartialField(args, "path"), toolcall.ReadPartialField(args, words.Match)
	if path == "" || match == "" {
		return
	}
	now := p.clock.Now()
	if now.Sub(p.last) < previewInterval {
		return
	}
	p.last = now
	p.emit(&agentrun.Response{Step: &agentrun.EditProgressStep{
		EditProgress: &agentrun.EditProgress{
			Call:  p.callID,
			Path:  path,
			Match: match,
			Text:  toolcall.ReadPartialField(args, words.Text),
		},
	}})
}

func getUnreachableReason(response event) string {
	for _, skipped := range response.ServerErrors {
		if skipped.Name == serverName {
			return fmt.Sprintf("the agent was not given this vault: %s", skipped.Message)
		}
	}
	if response.Servers == nil {
		return ""
	}
	for _, server := range *response.Servers {
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

func getResultError(response event) string {
	if text := strings.TrimSpace(response.Result); text != "" && text != response.Subtype {
		return text
	}
	if response.Subtype != "" {
		return formatSubtype(response.Subtype)
	}
	return "the agent stopped without finishing"
}

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
	case "error_interrupted", "error_canceled", "interrupted":
		return "the agent was interrupted"
	default:
		if rest, ok := strings.CutPrefix(subtype, "error_"); ok {
			return strings.ReplaceAll(rest, "_", " ")
		}
		return subtype
	}
}
