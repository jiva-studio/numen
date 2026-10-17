package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

const reportChunk = 200

type streamParser struct {
	stream      *connect.ServerStream[numenv1.AskAgentResponse]
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
	stream *connect.ServerStream[numenv1.AskAgentResponse],
	record func(string),
) string {
	p := &streamParser{
		stream: stream,
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
			_ = p.stream.Send(&numenv1.AskAgentResponse{
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
		case "status":
			if said.Status == "requesting" {
				_ = p.stream.Send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_ModelStarted{
						ModelStarted: &numenv1.ModelStarted{},
					},
				})
			}
		}
	case "result":
		if said.IsError {
			p.setReason(said.Result)
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
			_ = p.stream.Send(buildToolCallResponse(p.tool, ""))
		}
	case "content_block_delta":
		switch ev.Delta.Type {
		case "text_delta":
			if ev.Delta.Text == "" {
				return
			}
			p.isStreaming = true
			_ = p.stream.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Said{
					Said: ev.Delta.Text,
				},
			})
		case "input_json_delta":
			p.arguments.WriteString(ev.Delta.Partial)
			if p.arguments.Len()-p.offset >= reportChunk {
				p.offset = p.arguments.Len()
				_ = p.stream.Send(buildToolCallResponse(p.tool, p.arguments.String()))
			}
		}
	case "content_block_stop":
		if p.tool == "" {
			return
		}
		_ = p.stream.Send(buildToolCallResponse(p.tool, p.arguments.String()))
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
				_ = p.stream.Send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_Said{
						Said: block.Text,
					},
				})
			}
		case "tool_use":
			_ = p.stream.Send(buildToolCallResponse(block.Name, string(block.Input)))
		}
	}
}

func (p *streamParser) setReason(why string) {
	if p.reason == "" {
		p.reason = why
	}
}

func buildToolCallResponse(tool, args string) *numenv1.AskAgentResponse {
	toolName := tool
	if strings.HasPrefix(toolName, "mcp__numen__") {
		toolName = strings.TrimPrefix(toolName, "mcp__numen__")
	} else if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.SplitN(toolName, "__", 3)
		if len(parts) == 3 {
			toolName = parts[2]
		}
	}

	about, path, span := extractCallDetails(args)
	return &numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ToolCall{
			ToolCall: &numenv1.ToolCall{
				Tool:    toolName,
				About:   about,
				Written: int32(len(args)),
				Path:    path,
				Span:    span,
			},
		},
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
			var from, length int64
			if s, ok := startVal.(float64); ok {
				from = int64(s)
			}
			if l, ok := lenVal.(float64); ok {
				length = int64(l)
			}
			span = &numenv1.Span{
				From: int32(from),
				To:   int32(from + length),
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
