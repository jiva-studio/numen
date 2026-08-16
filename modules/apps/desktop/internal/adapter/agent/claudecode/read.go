package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"
)

// read turns what the agent prints into steps, and reports why it stopped when
// the stream says so.
//
// One JSON object arrives per line, but a line arrives in as many pieces as the
// pipe feels like: the tail of one read is the head of the next, so lines are
// put together before anything is decoded. A line that will not decode is
// passed over.
func read(ctx context.Context, r io.Reader, steps chan<- agent.Step, words map[string]Words, kept func(string)) string {
	reading := reader{steps: steps, words: words, kept: kept}
	lines := bufio.NewReader(r)

	for {
		line, err := lines.ReadString('\n')
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			reading.line(ctx, trimmed)
		}
		if err != nil {
			return reading.failed
		}
	}
}

// reader is what has been made of the stream so far.
type reader struct {
	steps chan<- agent.Step
	// words are what each tool this vault serves calls itself, and which
	// argument says what a call was about.
	words map[string]Words
	// kept is told which conversation this was, so that the next question can
	// be asked in the same one.
	kept func(string)
	// pieces is set once words have arrived a piece at a time. The whole
	// message follows every piece of it.
	pieces bool
	// calling is the tool being written out, and the arguments as far as they
	// have arrived. A call is reported once it is whole, so that what it is
	// about is known when it is shown.
	calling string
	written strings.Builder
	// failed is why the work stopped. The first reason is the one that holds.
	failed string
}

func (rd *reader) line(ctx context.Context, line string) {
	var said event
	if err := json.Unmarshal([]byte(line), &said); err != nil {
		return
	}

	switch said.Type {
	case "stream_event":
		rd.piece(ctx, said.Event)
	case "assistant":
		rd.whole(ctx, said)
	case "system":
		if said.Subtype == "init" {
			if said.Session != "" && rd.kept != nil {
				rd.kept(said.Session)
			}
			rd.stop(unreachable(said))
		}
	case "result":
		if said.IsError {
			rd.stop(result(said))
			return
		}
		rd.stop("")
	}
}

// piece reports a word as it is written, and a call once it is whole.
func (rd *reader) piece(ctx context.Context, event streamed) {
	switch event.Type {
	case "content_block_start":
		if event.Block.Type == "tool_use" {
			rd.pieces = true
			rd.calling = event.Block.Name
			rd.written.Reset()
		}
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			if event.Delta.Text == "" {
				return
			}
			rd.pieces = true
			rd.tell(ctx, agent.Step{Kind: agent.Saying, Text: event.Delta.Text})
		case "input_json_delta":
			rd.written.WriteString(event.Delta.Partial)
		}
	case "content_block_stop":
		if rd.calling == "" {
			return
		}
		rd.tell(ctx, rd.calls(rd.calling, rd.written.String()))
		rd.calling = ""
		rd.written.Reset()
	}
}

// whole reports a message that arrived in one piece, for a version that does
// not write them as they are made.
func (rd *reader) whole(ctx context.Context, said event) {
	if rd.pieces {
		return
	}
	for _, block := range said.blocks() {
		switch block.Type {
		case "text":
			if block.Text != "" {
				rd.tell(ctx, agent.Step{Kind: agent.Saying, Text: block.Text})
			}
		case "tool_use":
			rd.tell(ctx, rd.calls(block.Name, string(block.Input)))
		}
	}
}

// stop keeps the first reason the work ended.
func (rd *reader) stop(why string) {
	if rd.failed == "" {
		rd.failed = why
	}
}

// tell hands a step over, or gives up when nobody is listening any more.
func (rd *reader) tell(ctx context.Context, s agent.Step) {
	select {
	case rd.steps <- s:
	case <-ctx.Done():
	}
}

// unreachable is why this vault's tools did not arrive, empty when they did
// and empty when the line says nothing about servers at all.
func unreachable(said event) string {
	for _, skipped := range said.ServerErrors {
		if skipped.Name == Name {
			return fmt.Sprintf("the agent was not given this vault: %s", skipped.Message)
		}
	}
	if said.Servers == nil {
		return ""
	}
	for _, server := range *said.Servers {
		if server.Name != Name {
			continue
		}
		if server.Status == "failed" {
			return "the agent could not reach this vault"
		}
		return ""
	}
	return "the agent was started without this vault"
}

// result is what the last line says went wrong.
func result(said event) string {
	if text := strings.TrimSpace(said.Result); text != "" {
		return text
	}
	if said.Subtype != "" {
		return said.Subtype
	}
	return "the agent stopped without finishing"
}

// event is a line of what the agent prints, in the parts worth reading. The
// message is held as it arrived: what it carries depends on whose message it
// is.
type event struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	Session string          `json:"session_id"`
	Message json.RawMessage `json:"message"`
	Event   streamed        `json:"event"`
	Result  string          `json:"result"`
	IsError bool            `json:"is_error"`

	// Servers is absent from a line that says nothing about servers, and empty
	// on a line that says there are none.
	Servers *[]struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers"`
	ServerErrors []struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"mcp_server_errors"`
}

// streamed is a message being written: a block beginning, or a piece of one.
type streamed struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
		// Partial is the arguments of a call, arriving as text of JSON.
		Partial string `json:"partial_json"`
	} `json:"delta"`
	Block block `json:"content_block"`
}

// block is a piece of what the agent said: prose, or a tool it reached for.
type block struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// calling is a tool as the person is told about it: what the tool calls
// itself, and what this call was about.
//
// Both come from what the tool declared. The name is its title, and what the
// call is about is the argument it declared it cannot be called without. A
// tool this vault does not serve is named as it named itself and is about
// nothing: nothing was declared here to read it by.
func (rd *reader) calls(tool string, arguments string) agent.Step {
	words, served := rd.words[tool]
	if !served {
		return agent.Step{Kind: agent.Calling, Tool: tool}
	}

	step := agent.Step{Kind: agent.Calling, Tool: words.Title}
	if words.About == "" {
		return step
	}

	var made map[string]any
	if err := json.Unmarshal([]byte(arguments), &made); err != nil {
		return step
	}
	switch value := made[words.About].(type) {
	case string:
		step.About = value
	case []any:
		if len(value) == 0 {
			break
		}
		if first, ok := value[0].(string); ok {
			step.About = first
			if len(value) > 1 {
				step.About = fmt.Sprintf("%s and %d more", first, len(value)-1)
			}
		}
	}
	return step
}

func (e event) blocks() []block {
	var message struct {
		Content []block `json:"content"`
	}
	if err := json.Unmarshal(e.Message, &message); err != nil {
		return nil
	}
	return message.Content
}
