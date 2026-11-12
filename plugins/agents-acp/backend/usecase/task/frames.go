package task

import (
	"strconv"

	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/toolcall"
)

// sendError is a frame that could not be sent, which ends the task with the error of the send.
type sendError struct{ err error }

func (e *sendError) Error() string { return e.err.Error() }

func (e *sendError) Unwrap() error { return e.err }

// shownCall is what the frames of one tool call showed so far.
type shownCall struct {
	name       string
	title      string
	kind       contract.ToolKind
	isFinished bool
}

// turn turns the session updates of one prompt turn into the frames of the task, and allows the agent
// the tools the session serves.
type turn struct {
	listed       map[string]*tools.ToolInfo
	send         func(*contract.TaskFrame) error
	calls        map[string]*shownCall
	message      int
	isAfterChunk bool
}

func newTurn(listed map[string]*tools.ToolInfo, send func(*contract.TaskFrame) error) *turn {
	return &turn{listed: listed, send: send, calls: map[string]*shownCall{}, message: 1, isAfterChunk: false}
}

// IsAllowed reports whether the session serves the tool.
func (t *turn) IsAllowed(tool string) bool {
	_, isListed := t.listed[tool]
	return isListed
}

// HandleUpdate sends the frames of one session update.
func (t *turn) HandleUpdate(update acp.Update) error {
	switch {
	case update.Chunk != nil:
		return t.sendChunk(*update.Chunk)
	case update.Plan != nil:
		t.endMessage()
		return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_Plan{Plan: writePlan(*update.Plan)}})
	case update.Tool != nil:
		t.endMessage()
		return t.sendTool(*update.Tool)
	}
	return nil
}

// sendChunk sends a piece of the answer. Pieces the agent names no message for belong to one message until
// another update comes between them.
func (t *turn) sendChunk(chunk acp.Chunk) error {
	id := chunk.MessageID
	if id == "" {
		id = "m" + strconv.Itoa(t.message)
	}
	t.isAfterChunk = true
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_AgentMessageChunk{
		AgentMessageChunk: &contract.MessageChunk{MessageId: id, Text: chunk.Text},
	}})
}

func (t *turn) endMessage() {
	if t.isAfterChunk {
		t.message++
		t.isAfterChunk = false
	}
}

// sendTool sends the first report of a call as a tool_call and the others as tool_call_updates, and nothing
// after the call has finished. A call of a served tool shows the title of the listing, the kind of its
// effect, what it is about and the vault path it works on.
func (t *turn) sendTool(update acp.ToolUpdate) error {
	shown, isKnown := t.calls[update.ID]
	if !isKnown {
		shown = &shownCall{name: "", title: "", kind: contract.ToolKind_TOOL_KIND_UNSPECIFIED, isFinished: false}
		t.calls[update.ID] = shown
	}
	if shown.isFinished {
		return nil
	}
	if update.Name != "" {
		shown.name = update.Name
	}
	listed := t.listed[shown.name]
	title, kind := describeTool(listed, update)
	content, locations := describeArguments(listed, update)
	status := readStatus(update.Status)
	isFinal := status == contract.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED || status == contract.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	if !isKnown {
		if status == contract.ToolCallStatus_TOOL_CALL_STATUS_UNSPECIFIED {
			status = contract.ToolCallStatus_TOOL_CALL_STATUS_PENDING
		}
		shown.title, shown.kind = title, kind
		if err := t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCall{ToolCall: &contract.ToolCall{
			ToolCallId: update.ID, Title: title, Kind: kind, Status: status,
			Content: content, Locations: locations, RawInputJson: string(update.Arguments),
		}}}); err != nil {
			return err
		}
		if !isFinal {
			return nil
		}
	}
	return t.sendToolUpdate(shown, update, toolChange{title: title, kind: kind, status: status, content: content, locations: locations})
}

// toolChange is what an update of a call may change.
type toolChange struct {
	title     string
	kind      contract.ToolKind
	status    contract.ToolCallStatus
	content   []string
	locations []*contract.ToolCallLocation
}

func (t *turn) sendToolUpdate(shown *shownCall, update acp.ToolUpdate, change toolChange) error {
	frame := &contract.ToolCallUpdate{
		ToolCallId: update.ID, Title: nil, Kind: nil, Status: nil,
		Content: change.content, Locations: change.locations, RawOutputJson: string(update.Output),
	}
	if change.title != shown.title && change.title != "" {
		shown.title = change.title
		frame.Title = &change.title
	}
	if change.kind != shown.kind && change.kind != contract.ToolKind_TOOL_KIND_UNSPECIFIED {
		shown.kind = change.kind
		frame.Kind = &change.kind
	}
	if change.status != contract.ToolCallStatus_TOOL_CALL_STATUS_UNSPECIFIED {
		frame.Status = &change.status
	}
	shown.isFinished = change.status == contract.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED ||
		change.status == contract.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	return t.sendFrame(&contract.TaskFrame{Update: &contract.TaskFrame_ToolCallUpdate{ToolCallUpdate: frame}})
}

func (t *turn) sendFrame(frame *contract.TaskFrame) error {
	if err := t.send(frame); err != nil {
		return &sendError{err: err}
	}
	return nil
}

// describeTool returns the title and the kind of a call: those of the listing for a served tool, else
// those the agent reports.
func describeTool(listed *tools.ToolInfo, update acp.ToolUpdate) (string, contract.ToolKind) {
	if listed != nil {
		title := listed.GetTitle()
		if title == "" {
			title = listed.GetName()
		}
		if listed.GetEffect() == tools.ReadWrite {
			return title, contract.ToolKind_TOOL_KIND_EDIT
		}
		return title, contract.ToolKind_TOOL_KIND_READ
	}
	title := ""
	if update.Title != nil {
		title = *update.Title
	}
	kind := contract.ToolKind_TOOL_KIND_UNSPECIFIED
	if update.Kind != nil {
		kind = readKind(*update.Kind)
	}
	return title, kind
}

// describeArguments returns what the call is about as the first content, and, for a served tool, the vault
// path it works on. Both are empty where the update holds no arguments.
func describeArguments(listed *tools.ToolInfo, update acp.ToolUpdate) ([]string, []*contract.ToolCallLocation) {
	if len(update.Arguments) == 0 {
		return nil, nil
	}
	about, path := toolcall.ReadSubject(update.Arguments)
	var locations []*contract.ToolCallLocation
	if listed != nil && path != "" {
		locations = []*contract.ToolCallLocation{{Path: path, Line: 0}}
	}
	if about == "" {
		return nil, locations
	}
	return []string{about}, locations
}

func readKind(kind string) contract.ToolKind {
	switch kind {
	case "read":
		return contract.ToolKind_TOOL_KIND_READ
	case "edit":
		return contract.ToolKind_TOOL_KIND_EDIT
	case "delete":
		return contract.ToolKind_TOOL_KIND_DELETE
	case "move":
		return contract.ToolKind_TOOL_KIND_MOVE
	case "search":
		return contract.ToolKind_TOOL_KIND_SEARCH
	case "execute":
		return contract.ToolKind_TOOL_KIND_EXECUTE
	case "think":
		return contract.ToolKind_TOOL_KIND_THINK
	case "fetch":
		return contract.ToolKind_TOOL_KIND_FETCH
	}
	return contract.ToolKind_TOOL_KIND_OTHER
}

func readStatus(status *string) contract.ToolCallStatus {
	if status == nil {
		return contract.ToolCallStatus_TOOL_CALL_STATUS_UNSPECIFIED
	}
	switch *status {
	case "pending":
		return contract.ToolCallStatus_TOOL_CALL_STATUS_PENDING
	case "in_progress":
		return contract.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
	case "completed":
		return contract.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	case "failed":
		return contract.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	}
	return contract.ToolCallStatus_TOOL_CALL_STATUS_UNSPECIFIED
}

func writePlan(plan acp.Plan) *contract.Plan {
	entries := make([]*contract.PlanEntry, 0, len(plan.Entries))
	for _, one := range plan.Entries {
		entries = append(entries, &contract.PlanEntry{Content: one.Content, Priority: readPriority(one.Priority), Status: readPlanStatus(one.Status)})
	}
	return &contract.Plan{Entries: entries}
}

func readPriority(priority string) contract.PlanEntryPriority {
	switch priority {
	case "high":
		return contract.PlanEntryPriority_PLAN_ENTRY_PRIORITY_HIGH
	case "medium":
		return contract.PlanEntryPriority_PLAN_ENTRY_PRIORITY_MEDIUM
	case "low":
		return contract.PlanEntryPriority_PLAN_ENTRY_PRIORITY_LOW
	}
	return contract.PlanEntryPriority_PLAN_ENTRY_PRIORITY_UNSPECIFIED
}

func readPlanStatus(status string) contract.PlanEntryStatus {
	switch status {
	case "pending":
		return contract.PlanEntryStatus_PLAN_ENTRY_STATUS_PENDING
	case "in_progress":
		return contract.PlanEntryStatus_PLAN_ENTRY_STATUS_IN_PROGRESS
	case "completed":
		return contract.PlanEntryStatus_PLAN_ENTRY_STATUS_COMPLETED
	}
	return contract.PlanEntryStatus_PLAN_ENTRY_STATUS_UNSPECIFIED
}

func writeStopReason(reason acp.StopReason) contract.StopReason {
	switch reason {
	case acp.StopEndTurn:
		return contract.StopReason_STOP_REASON_END_TURN
	case acp.StopMaxTokens:
		return contract.StopReason_STOP_REASON_MAX_TOKENS
	case acp.StopMaxTurnRequests:
		return contract.StopReason_STOP_REASON_MAX_TURN_REQUESTS
	case acp.StopRefusal:
		return contract.StopReason_STOP_REASON_REFUSAL
	case acp.StopCancelled:
		return contract.StopReason_STOP_REASON_CANCELLED
	}
	return contract.StopReason_STOP_REASON_UNSPECIFIED
}
