package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// read turns what the agent prints into steps, and reports why it stopped when
// the stream says so.
//
// One JSON object arrives per line, but a line arrives in as many pieces as the
// pipe feels like: the tail of one read is the head of the next, so lines are
// put together before anything is decoded. A line that will not decode is
// passed over.
func read(
	ctx context.Context,
	r io.Reader,
	steps chan<- port.Step,
	words map[string]Words,
	kept func(string),
	draft Drafting,
) string {
	reading := reader{steps: steps, words: words, kept: kept, draft: draft}
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

// writtenStep is how much more of a call has to be written before it is
// reported again.
const writtenStep = 200

// framePace is how often a change being written is drawn. The words arrive
// faster than a screen is redrawn.
const framePace = 50 * time.Millisecond

// reader is what has been made of the stream so far.
type reader struct {
	steps chan<- port.Step
	// words are what each tool this vault serves calls itself, and which
	// argument says what a call was about.
	words map[string]Words
	// kept is told which session this run is on, so that the next question of
	// the same conversation is asked in it.
	kept func(string)
	// pieces is set once words have arrived a piece at a time. The whole
	// message follows every piece of it.
	pieces bool
	// call is what the agent named the call being written. Every step of that
	// call carries it.
	call string
	// calling is the tool being written out, and the arguments as far as they
	// have arrived. A call is reported once it is whole, so that what it is
	// about is known when it is shown.
	calling string
	written strings.Builder
	// told is how much of the call had been reported the last time it was.
	told int
	// draft is how a change being written is drawn before it lands.
	draft Drafting
	// path is the note the change being written goes into, and from and to are
	// the stretch it replaces. Set once that stretch has been found.
	path     string
	from, to int
	drawn    bool
	// at is when a frame was last sent. A long call is drawn at a pace a screen
	// can keep.
	at time.Time
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
	case "user":
		// The tool answered. Everything from here until the next block arrives
		// is the model's, and the step says so.
		if said.answers() {
			rd.tell(ctx, port.Step{Kind: port.StepAnswered})
		}
	case "system":
		switch said.Subtype {
		case "init":
			if said.Session != "" && rd.kept != nil {
				rd.kept(said.Session)
			}
			rd.stop(unreachable(said))
		case "status":
			// `requesting` is written when a request to the model begins. It is
			// the start of the wait, reported by the agent itself.
			if said.Status == "requesting" {
				rd.tell(ctx, port.Step{Kind: port.StepThinking})
			}
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
			rd.call = event.Block.ID
			rd.calling = event.Block.Name
			rd.written.Reset()
			rd.told = 0
			rd.path, rd.from, rd.to, rd.drawn = "", 0, 0, false
			rd.at = time.Time{}
			rd.tell(ctx, rd.calls(rd.call, rd.calling, ""))
		}
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			if event.Delta.Text == "" {
				return
			}
			rd.pieces = true
			rd.tell(ctx, port.Step{Kind: port.StepSaying, Text: event.Delta.Text})
		case "input_json_delta":
			rd.written.WriteString(event.Delta.Partial)
			// Reported as it is written, one writtenStep of characters at a
			// time. A call carrying the body of a note is written for minutes.
			if rd.written.Len()-rd.told >= writtenStep {
				rd.told = rd.written.Len()
				rd.tell(ctx, rd.calls(rd.call, rd.calling, rd.written.String()))
			}
			rd.draw(ctx)
		}
	case "content_block_stop":
		if rd.calling == "" {
			return
		}
		rd.tell(ctx, rd.calls(rd.call, rd.calling, rd.written.String()))
		rd.call = ""
		rd.calling = ""
		rd.written.Reset()
		rd.drawn = false
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
				rd.tell(ctx, port.Step{Kind: port.StepSaying, Text: block.Text})
			}
		case "tool_use":
			rd.tell(ctx, rd.calls(block.ID, block.Name, string(block.Input)))
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
func (rd *reader) tell(ctx context.Context, s port.Step) {
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

	// Status is what a status line says is happening now.
	Status string `json:"status"`

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
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
	// ID is what the agent named a call, and is carried by every report of it.
	ID    string          `json:"id"`
	Input json.RawMessage `json:"input"`
}

// notePath is the argument a tool of this vault names one note by. A call read
// by it is about a path, and that path is where the call is working.
const notePath = "path"

// spanStart and spanLength are the arguments a tool of this vault names a
// stretch of a source's text by.
const (
	spanStart  = "start"
	spanLength = "length"
)

// calls is a tool as the person is told about it: what the tool calls itself,
// what this call was about, what it does to the vault, and what the agent named
// the call.
//
// All but the name come from what the tool declared. The name it is shown by is
// its title, and what the call is about is the argument it declared it cannot be
// called without. A tool this vault does not serve is named as it named itself
// and is about nothing: nothing was declared here to read it by.
func (rd *reader) calls(call, tool, arguments string) port.Step {
	words, served := rd.words[tool]
	if !served {
		return port.Step{Kind: port.StepCalling, Call: call, Tool: tool}
	}

	step := port.Step{Kind: words.Kind, Call: call, Tool: words.Title}
	if words.About == "" {
		return step
	}

	step.Written = len([]rune(arguments))
	step.About = about(words, arguments)
	if words.About == notePath {
		step.Place = placed(step.About, arguments)
	}
	return step
}

// placed is where a call is working: the path it named, and the stretch of that
// source's text it named beside it.
//
// The stretch is read once the arguments parse whole, so it arrives with the
// report that ends the call. Half a number is another number.
func placed(path, arguments string) domain.Place {
	at := domain.Place{Path: path}
	var made map[string]any
	if err := json.Unmarshal([]byte(arguments), &made); err != nil {
		return at
	}
	start, _ := made[spanStart].(float64)
	length, _ := made[spanLength].(float64)
	at.Start, at.Length = int(start), int(length)
	return at
}

// about is what a call was about, read from the arguments as far as they have
// arrived.
//
// Arguments still arriving is where most of a long wait is spent, and half a
// document does not parse. What has been written is read for the name, so that
// the person sees which note is being written while it is being written.
func about(words Words, arguments string) string {
	var made map[string]any
	if err := json.Unmarshal([]byte(arguments), &made); err != nil {
		if seen := glimpsed(arguments, words.Inside); seen != "" {
			return seen
		}
		return glimpsed(arguments, words.About)
	}
	switch value := made[words.About].(type) {
	case string:
		return value
	case []any:
		return named(value, words.Inside)
	}
	return ""
}

// named is what a collection of arguments is about: the first element by the
// name it carries, and how many others there are.
func named(value []any, inside string) string {
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

// glimpsed is the value of a named field in JSON that has not finished
// arriving.
//
// Nothing here decodes: half a document does not parse, and waiting for the
// whole of it is waiting for the thing being watched. The first field of that
// name is taken, since the first element of a collection is the one being
// written when there is nothing else to show yet.
func glimpsed(arguments, field string) string {
	if field == "" {
		return ""
	}
	at := strings.Index(arguments, `"`+field+`"`)
	if at < 0 {
		return ""
	}
	rest := arguments[at+len(field)+2:]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]

	// One way out, so that every value read is read the same way. An escape whose
	// second half has not arrived, and a character cut in half, both end the
	// value where they begin.
	var out strings.Builder
	for i := 0; i < len(rest); i++ {
		if rest[i] == '"' {
			break
		}
		if rest[i] == '\\' {
			if i+1 >= len(rest) {
				break
			}
			out.WriteByte(rest[i])
			i++
		}
		out.WriteByte(rest[i])
	}
	return unquoted(whole(out.String()))
}

// whole is the text without a character that has half arrived.
//
// Pieces are cut where the stream cut them, which for anything outside ASCII is
// as likely to be the middle of a character as the end of one.
func whole(text string) string {
	for len(text) > 0 {
		last, size := utf8.DecodeLastRuneInString(text)
		if last != utf8.RuneError || size > 1 {
			return text
		}
		text = text[:len(text)-1]
	}
	return text
}

// unquoted turns the escapes of a JSON string into what they stand for.
//
// The last escape may have arrived in pieces — a character named by number is
// six of them, and five are not a character — so the tail is given up a piece at
// a time until what is left reads. An escape shown as itself is text the person
// did not write.
func unquoted(text string) string {
	for at := len(text); at > 0; at-- {
		var out string
		if err := json.Unmarshal([]byte(`"`+text[:at]+`"`), &out); err == nil {
			return out
		}
	}
	return ""
}

// answers reports whether this line carries the answer of a tool.
func (e event) answers() bool {
	for _, block := range e.blocks() {
		if block.Type == "tool_result" {
			return true
		}
	}
	return false
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

// draw reports what a change is doing while the call making it is still being
// written.
//
// The text going in arrives after the text it replaces, so nothing is drawn
// until the replacement has begun: a stretch shown with nothing in its place
// reads as having been deleted. Once the replacement has begun the stretch it
// replaces is whole, and where it stands can be found.
func (rd *reader) draw(ctx context.Context) {
	words, served := rd.words[rd.calling]
	if !served || words.Becomes == "" || !rd.draft.drawing() {
		return
	}

	arguments := rd.written.String()
	if !strings.Contains(arguments, `"`+words.Becomes+`"`) {
		return
	}
	if !rd.drawn {
		path, stood := glimpsed(arguments, words.About), glimpsed(arguments, words.Stood)
		if path == "" || stood == "" {
			return
		}
		from, to, one := rd.draft.Where(ctx, path, stood)
		if !one {
			return
		}
		rd.path, rd.from, rd.to, rd.drawn = path, from, to, true
	}

	now := rd.draft.now()
	if now.Sub(rd.at) < framePace {
		return
	}
	rd.at = now
	rd.draft.Tell(ctx, domain.Editing{
		Change: rd.call,
		Path:   rd.path,
		From:   rd.from,
		To:     rd.to,
		Text:   glimpsed(arguments, words.Becomes),
	})
}
