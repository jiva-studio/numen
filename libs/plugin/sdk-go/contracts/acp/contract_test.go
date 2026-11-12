package acp_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
)

// inProcess is a connection.Caller that runs the operations of a provider in the test.
type inProcess struct{ provider connection.Provider }

func (c inProcess) Call(ctx context.Context, operation string, payload []byte, _ connection.Priority) (connection.Frames, error) {
	got := &frames{}
	got.done, got.err = c.provider[operation](ctx, connection.Call{Type: "claude-code"}, payload, func(data []byte) error {
		got.data = append(got.data, data)
		return nil
	})
	return got, nil
}

type frames struct {
	data [][]byte
	next int
	done []byte
	err  error
}

func (f *frames) Next() bool {
	if f.next >= len(f.data) {
		return false
	}
	f.next++
	return true
}

func (f *frames) Data() []byte { return f.data[f.next-1] }

func (f *frames) Done() []byte { return f.done }

func (f *frames) Err() error { return f.err }

func (f *frames) Close() error { return nil }

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
	caller := inProcess{provider: acp.Provide(nil, runner)}

	stream, err := acp.RunTask(t.Context(), caller, &acp.TaskRequest{
		Prompt:       "When is the Lisbon trip?",
		ContextNotes: []*acp.ContextNote{{Path: "Trips/Lisbon.md", Title: "Lisbon"}},
		Conversation: "conversation-7",
		ToolContext:  map[string]string{"org.example.trips/trip": "lisbon"},
	}, connection.PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	var text string
	for stream.Next() {
		switch update := stream.Frame().GetUpdate().(type) {
		case *acp.TaskFrame_Plan:
			kinds = append(kinds, "plan")
		case *acp.TaskFrame_ToolCall:
			kinds = append(kinds, "tool call "+update.ToolCall.GetTitle())
		case *acp.TaskFrame_ToolCallUpdate:
			kinds = append(kinds, "update "+update.ToolCallUpdate.GetStatus().String())
		case *acp.TaskFrame_AgentMessageChunk:
			text += update.AgentMessageChunk.GetText()
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"plan", "tool call Read Trips/Lisbon.md", "update TOOL_CALL_STATUS_COMPLETED"}
	if len(kinds) != len(want) || kinds[0] != want[0] || kinds[1] != want[1] || kinds[2] != want[2] || text != "Lisbon is in May." {
		t.Errorf("updates = %v, text = %q", kinds, text)
	}
	if stream.Done().GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Errorf("done = %v", stream.Done())
	}
	if runner.request.GetPrompt() != "When is the Lisbon trip?" || runner.request.GetContextNotes()[0].GetPath() != "Trips/Lisbon.md" ||
		runner.request.GetConversation() != "conversation-7" || runner.request.GetToolContext()["org.example.trips/trip"] != "lisbon" {
		t.Errorf("request = %v", runner.request)
	}
}

func TestAPayloadThatIsNoTaskRequestIsAnInvalidRequest(t *testing.T) {
	serve := acp.Provide(nil, &planner{})[acp.OperationTask]

	_, err := serve(t.Context(), connection.Call{}, []byte{0xff, 0xff}, func([]byte) error { return nil })

	if reason, _ := connection.GetReason(err); !connection.IsStop(err) || reason != connection.ReasonInvalidRequest {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}
