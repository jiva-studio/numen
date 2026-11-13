package task

import (
	"strconv"

	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/codexevent"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/toolcall"
)

// shownCall is what the frames of one tool call showed so far.
type shownCall struct {
	title      string
	kind       contract.ToolKind
	isFinished bool
}

// turn turns what one run of the program reports into the frames of the task. The text between two
// calls is one message. The first report of a call is sent as a tool_call, the others as
// tool_call_updates, and nothing after the call has finished. A call of a served tool shows the title of
// the listing, the kind of its effect, what it is about and the vault path it works on; any other call
// shows its name. The reasoning of the model is sent as thought chunks. thread is the thread the program
// reported, and isSent tells whether any frame was sent.
type turn struct {
	listed       map[string]*tools.ToolInfo
	send         func(*contract.TaskFrame) error
	calls        map[string]*shownCall
	message      int
	isAfterChunk bool
	sendErr      error
	thread       string
	isSent       bool
}

var _ codexevent.Sink = (*turn)(nil)

func newTurn(listed map[string]*tools.ToolInfo, send func(*contract.TaskFrame) error) *turn {
	return &turn{
		listed: listed, send: send, calls: map[string]*shownCall{}, message: 1, isAfterChunk: false, sendErr: nil,
		thread: "", isSent: false,
	}
}

// SendText sends a piece of the answer as a chunk of the current message.
func (t *turn) SendText(text string) error {
	t.isAfterChunk = true
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_AgentMessageChunk{
		AgentMessageChunk: &contract.MessageChunk{MessageId: "m" + strconv.Itoa(t.message), Text: text},
	}})
}

// RecordThread keeps the thread the program works in.
func (t *turn) RecordThread(id string) {
	t.thread = id
}

// SendThought sends a piece of the reasoning of the model as a thought chunk.
func (t *turn) SendThought(text string) error {
	t.endMessage()
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_AgentThoughtChunk{
		AgentThoughtChunk: &contract.MessageChunk{MessageId: "t" + strconv.Itoa(t.message), Text: text},
	}})
}

// SendMessage ends the current message and sends text as a message of its own, so the next text starts
// another.
func (t *turn) SendMessage(text string) error {
	t.endMessage()
	if err := t.SendText(text); err != nil {
		return err
	}
	t.endMessage()
	return nil
}

func (t *turn) endMessage() {
	if t.isAfterChunk {
		t.message++
		t.isAfterChunk = false
	}
}

// SendToolCall sends the start of a call, or what its arguments say once they are whole.
func (t *turn) SendToolCall(id, tool, arguments string) error {
	t.endMessage()
	shown, isKnown := t.calls[id]
	if isKnown && shown.isFinished {
		return nil
	}
	listed := t.listed[tool]
	title, kind := describeTool(listed, tool)
	content, locations := describeArguments(listed, arguments)
	if !isKnown {
		t.calls[id] = &shownCall{title: title, kind: kind, isFinished: false}
		status := contract.ToolCallStatus_TOOL_CALL_STATUS_PENDING
		if arguments != "" {
			status = contract.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
		}
		return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCall{ToolCall: &contract.ToolCall{
			ToolCallId: id, Title: title, Kind: kind, Status: status,
			Content: content, Locations: locations, RawInputJson: arguments,
		}}})
	}
	status := contract.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCallUpdate{ToolCallUpdate: &contract.ToolCallUpdate{
		ToolCallId: id, Title: nil, Kind: nil, Status: &status, Content: content, Locations: locations, RawOutputJson: "", Diffs: nil,
	}}})
}

// SendToolResult sends the end of a call that was shown.
func (t *turn) SendToolResult(id string, isFailed bool) error {
	t.endMessage()
	shown, isKnown := t.calls[id]
	if !isKnown || shown.isFinished {
		return nil
	}
	shown.isFinished = true
	status := contract.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	if isFailed {
		status = contract.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	}
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCallUpdate{ToolCallUpdate: &contract.ToolCallUpdate{
		ToolCallId: id, Title: nil, Kind: nil, Status: &status, Content: nil, Locations: nil, RawOutputJson: "", Diffs: nil,
	}}})
}

func (t *turn) sendFrame(frame *contract.TaskFrame) error {
	t.isSent = true
	if err := t.send(frame); err != nil {
		t.sendErr = err
		return err
	}
	return nil
}

func describeTool(listed *tools.ToolInfo, name string) (string, contract.ToolKind) {
	if listed == nil {
		return name, contract.ToolKind_TOOL_KIND_OTHER
	}
	title := listed.GetTitle()
	if title == "" {
		title = listed.GetName()
	}
	if listed.GetEffect() == tools.ReadWrite {
		return title, contract.ToolKind_TOOL_KIND_EDIT
	}
	return title, contract.ToolKind_TOOL_KIND_READ
}

func describeArguments(listed *tools.ToolInfo, arguments string) ([]string, []*contract.ToolCallLocation) {
	if arguments == "" {
		return nil, nil
	}
	about, path := toolcall.ReadSubject([]byte(arguments))
	var locations []*contract.ToolCallLocation
	if listed != nil && path != "" {
		locations = []*contract.ToolCallLocation{{Path: path, Line: 0}}
	}
	if about == "" {
		return nil, locations
	}
	return []string{about}, locations
}
