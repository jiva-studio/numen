package task

import (
	"strconv"

	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/toolcall"
)

// turn turns what one run of the program reports into the frames of the task. The text between two
// calls is one message. The program names no call and reports no end of one: each call it reports is
// sent as a tool_call of an id of its own, and the calls end with the run. A call of a served tool shows
// the title of the listing, the kind of its effect, what it is about and the vault path it works on; any
// other call shows its name.
type turn struct {
	listed       map[string]*tools.ToolInfo
	send         func(*contract.TaskFrame) error
	open         []string
	calls        int
	message      int
	isAfterChunk bool
	sendErr      error
}

func newTurn(listed map[string]*tools.ToolInfo, send func(*contract.TaskFrame) error) *turn {
	return &turn{listed: listed, send: send, open: nil, calls: 0, message: 1, isAfterChunk: false, sendErr: nil}
}

// SendText sends a piece of the answer as a chunk of the current message.
func (t *turn) SendText(text string) error {
	t.isAfterChunk = true
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_AgentMessageChunk{
		AgentMessageChunk: &contract.MessageChunk{MessageId: "m" + strconv.Itoa(t.message), Text: text},
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

// SendToolCall sends a call the program reported with its whole arguments.
func (t *turn) SendToolCall(tool, arguments string) error {
	t.endMessage()
	t.calls++
	id := "c" + strconv.Itoa(t.calls)
	t.open = append(t.open, id)
	listed := t.listed[tool]
	title, kind := describeTool(listed, tool)
	content, locations := describeArguments(listed, arguments)
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCall{ToolCall: &contract.ToolCall{
		ToolCallId: id, Title: title, Kind: kind, Status: contract.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS,
		Content: content, Locations: locations, RawInputJson: arguments,
	}}})
}

func (t *turn) finishCalls() error {
	status := contract.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	for _, id := range t.open {
		if err := t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCallUpdate{ToolCallUpdate: &contract.ToolCallUpdate{
			ToolCallId: id, Title: nil, Kind: nil, Status: &status, Content: nil, Locations: nil, RawOutputJson: "",
		}}}); err != nil {
			return err
		}
	}
	t.open = nil
	return nil
}

func (t *turn) sendFrame(frame *contract.TaskFrame) error {
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
