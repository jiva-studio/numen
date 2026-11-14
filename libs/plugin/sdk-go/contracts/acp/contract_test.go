package acp_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp/acptest"
)

// planner answers a prompt with a thought, a plan, a tool call, an update with the note it is writing, its
// end, and a chunk of text.
type planner struct{ request *acp.TaskRequest }

func (p *planner) RunTask(_ context.Context, _ connection.Call, request *acp.TaskRequest, send func(*acp.TaskFrame) error) (*acp.TaskDone, error) {
	p.request = request
	completed := acp.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	for _, frame := range []*acp.TaskFrame{
		{Update: &acp.TaskFrame_AgentThoughtChunk{AgentThoughtChunk: &acp.MessageChunk{MessageId: "t1", Text: "The trip note has the dates."}}},
		{Update: &acp.TaskFrame_Plan{Plan: &acp.Plan{Entries: []*acp.PlanEntry{{Content: "Read the trip notes", Status: acp.PlanEntryStatus_PLAN_ENTRY_STATUS_PENDING}}}}},
		{Update: &acp.TaskFrame_ToolCall{ToolCall: &acp.ToolCall{ToolCallId: "call-1", Title: "Read Trips/Lisbon.md", Kind: acp.ToolKind_TOOL_KIND_READ}}},
		{Update: &acp.TaskFrame_ToolCallUpdate{ToolCallUpdate: &acp.ToolCallUpdate{
			ToolCallId: "call-1", Diffs: []*acp.Diff{{Path: "Trips/Lisbon.md", NewText: "# Lisbon\n\nIn May"}},
		}}},
		{Update: &acp.TaskFrame_ToolCallUpdate{ToolCallUpdate: &acp.ToolCallUpdate{ToolCallId: "call-1", Status: &completed}}},
		{Update: &acp.TaskFrame_AgentMessageChunk{AgentMessageChunk: &acp.MessageChunk{MessageId: "m1", Text: "Lisbon is in May."}}},
	} {
		if err := send(frame); err != nil {
			return nil, err
		}
	}
	return &acp.TaskDone{StopReason: acp.StopReason_STOP_REASON_END_TURN}, nil
}

func TestATaskStreamsTheSessionUpdatesOfTheTurnAndEndsWithTheStopReason(t *testing.T) {
	runner := &planner{}
	var kinds []string
	var text string

	done, err := acptest.RunTask(t.Context(), acp.Provide(nil, runner), connection.Call{Type: "claude-code"}, &acp.TaskRequest{
		Prompt:       "When is the Lisbon trip?",
		ContextNotes: []*acp.ContextNote{{Path: "Trips/Lisbon.md", Title: "Lisbon"}},
		Conversation: "conversation-7",
		ToolContext:  map[string]string{"org.example.trips/trip": "lisbon"},
	}, func(frame *acp.TaskFrame) error {
		switch update := frame.GetUpdate().(type) {
		case *acp.TaskFrame_AgentThoughtChunk:
			kinds = append(kinds, "thought "+update.AgentThoughtChunk.GetText())
		case *acp.TaskFrame_Plan:
			kinds = append(kinds, "plan")
		case *acp.TaskFrame_ToolCall:
			kinds = append(kinds, "tool call "+update.ToolCall.GetTitle())
		case *acp.TaskFrame_ToolCallUpdate:
			change := update.ToolCallUpdate
			if diffs := change.GetDiffs(); len(diffs) > 0 {
				kinds = append(kinds, "diff "+diffs[0].GetPath()+" "+diffs[0].GetNewText())
			} else {
				kinds = append(kinds, "update "+change.GetStatus().String())
			}
		case *acp.TaskFrame_AgentMessageChunk:
			text += update.AgentMessageChunk.GetText()
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"thought The trip note has the dates.", "plan", "tool call Read Trips/Lisbon.md", "diff Trips/Lisbon.md # Lisbon\n\nIn May",
		"update TOOL_CALL_STATUS_COMPLETED",
	}
	if !slices.Equal(kinds, want) || text != "Lisbon is in May." {
		t.Errorf("updates = %v, text = %q", kinds, text)
	}
	if done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Errorf("done = %v", done)
	}
	if runner.request.GetPrompt() != "When is the Lisbon trip?" || runner.request.GetContextNotes()[0].GetPath() != "Trips/Lisbon.md" ||
		runner.request.GetConversation() != "conversation-7" || runner.request.GetToolContext()["org.example.trips/trip"] != "lisbon" {
		t.Errorf("request = %v", runner.request)
	}
}

func TestAPayloadThatIsNoTaskRequestIsAnInvalidRequest(t *testing.T) {
	serve := acp.Provide(nil, &planner{})[acp.OperationTask]

	_, err := serve(t.Context(), connection.Call{}, []byte{0xff, 0xff}, func([]byte) error { return nil })

	if !connection.IsStop(err) || err.Error() != connection.ReasonInvalidRequest.String() {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}
