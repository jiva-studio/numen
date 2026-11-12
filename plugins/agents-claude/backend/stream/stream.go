// Package stream reads the output of a Claude Code run, in the stream-json format of the program, and
// hands what the agent says and the tools it calls to a sink.
package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
)

// serverName is what the vault's tool server calls itself, and the prefix its tools arrive under.
const serverName = "numen"

// reasonCanceled is the reason of a run whose context ended before its output did.
const reasonCanceled = "canceled"

// Sink receives what a run reports, in the order the program writes it. A call is reported once when
// it starts, with no arguments when they are still to arrive, and again once its arguments are whole.
type Sink interface {
	SendText(text string) error
	SendToolCall(id, tool, arguments string) error
	SendToolResult(id string, isFailed bool) error
}

// Options are what the parser of a stream needs besides the stream and the sink. Record receives the
// session the program reports, and ExpectsVault makes the parser report a program that started without
// the vault.
type Options struct {
	Record       func(string)
	ExpectsVault bool
}

type streamParser struct {
	sink         Sink
	record       func(string)
	isStreaming  bool
	callID       string
	tool         string
	arguments    strings.Builder
	reason       string
	expectsVault bool
	sendErr      error
}

func (p *streamParser) emit(send func() error) {
	if p.sendErr == nil {
		p.sendErr = send()
	}
}

// Parse reads the output of the program line by line and hands each report to the sink. It returns the
// reason the run failed, or none, once the output ends or ctx is done, and the error of the first
// report the sink refused. A sink that refused a report is handed no other.
func Parse(ctx context.Context, r io.Reader, sink Sink, opts Options) (string, error) {
	p := &streamParser{
		sink: sink, record: opts.Record, expectsVault: opts.ExpectsVault,
		isStreaming: false, callID: "", tool: "", arguments: strings.Builder{}, reason: "", sendErr: nil,
	}
	scanner := bufio.NewReader(r)
	for {
		if err := ctx.Err(); err != nil {
			return reasonCanceled, p.sendErr
		}
		line, err := scanner.ReadString('\n')
		if content := strings.TrimSpace(line); content != "" {
			p.parseLine(content)
		}
		if err != nil {
			return p.reason, p.sendErr
		}
	}
}

func (p *streamParser) parseLine(line string) {
	var response event
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return
	}
	switch response.Type {
	case "stream_event":
		p.parseStreamEvent(response.Event)
	case "assistant":
		p.parseAssistantMessage(response)
	case "user":
		for _, answer := range response.getToolAnswers() {
			p.emit(func() error { return p.sink.SendToolResult(answer.ID, answer.IsError) })
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
	if response.Subtype != "init" {
		return
	}
	if response.Session != "" && p.record != nil {
		p.record(response.Session)
	}
	if p.expectsVault {
		p.setReason(getUnreachableReason(response))
	}
}

func (p *streamParser) parseStreamEvent(ev streamEvent) {
	switch ev.Type {
	case "content_block_start":
		if ev.Block.Type == "tool_use" {
			p.isStreaming = true
			p.callID = ev.Block.ID
			p.tool = ev.Block.Name
			p.arguments.Reset()
			id, tool := p.callID, stripServerPrefix(p.tool)
			p.emit(func() error { return p.sink.SendToolCall(id, tool, "") })
		}
	case "content_block_delta":
		switch ev.Delta.Type {
		case "text_delta":
			if ev.Delta.Text == "" {
				return
			}
			p.isStreaming = true
			p.emit(func() error { return p.sink.SendText(ev.Delta.Text) })
		case "input_json_delta":
			p.arguments.WriteString(ev.Delta.Partial)
		}
	case "content_block_stop":
		if p.tool == "" {
			return
		}
		id, tool, arguments := p.callID, stripServerPrefix(p.tool), p.arguments.String()
		p.emit(func() error { return p.sink.SendToolCall(id, tool, arguments) })
		p.callID = ""
		p.tool = ""
		p.arguments.Reset()
	}
}

func (p *streamParser) parseAssistantMessage(response event) {
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
				p.emit(func() error { return p.sink.SendText(block.Text) })
			}
		case "tool_use":
			p.emit(func() error { return p.sink.SendToolCall(block.ID, stripServerPrefix(block.Name), string(block.Input)) })
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
