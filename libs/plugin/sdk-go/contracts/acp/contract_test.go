package acp_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp/acptest"
)

// planner answers a prompt with a plan, a tool call and its update, and a chunk of text.
type planner struct{ request *acp.TaskRequest }

func (p *planner) RunTask(_ context.Context, _ connection.Call, request *acp.TaskRequest, send func(*acp.TaskFrame) error) (*acp.TaskDone, error) {
	p.request = request
	completed := acp.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	for _, frame := range []*acp.TaskFrame{
		{Update: &acp.TaskFrame_Plan{Plan: &acp.Plan{Entries: []*acp.PlanEntry{{Content: "Read the trip notes", Status: acp.PlanEntryStatus_PLAN_ENTRY_STATUS_PENDING}}}}},
		{Update: &acp.TaskFrame_ToolCall{ToolCall: &acp.ToolCall{ToolCallId: "call-1", Title: "Read Trips/Lisbon.md", Kind: acp.ToolKind_TOOL_KIND_READ}}},
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
		case *acp.TaskFrame_Plan:
			kinds = append(kinds, "plan")
		case *acp.TaskFrame_ToolCall:
			kinds = append(kinds, "tool call "+update.ToolCall.GetTitle())
		case *acp.TaskFrame_ToolCallUpdate:
			kinds = append(kinds, "update "+update.ToolCallUpdate.GetStatus().String())
		case *acp.TaskFrame_AgentMessageChunk:
			text += update.AgentMessageChunk.GetText()
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	want := []string{"plan", "tool call Read Trips/Lisbon.md", "update TOOL_CALL_STATUS_COMPLETED"}
	if len(kinds) != len(want) || kinds[0] != want[0] || kinds[1] != want[1] || kinds[2] != want[2] || text != "Lisbon is in May." {
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
