package acp_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

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

	sendErr := errors.New("send failed")
	if _, err := serve(t.Context(), connection.Call{}, nil, func([]byte) error { return sendErr }); !errors.Is(err, sendErr) {
		t.Errorf("err = %v, want %v", err, sendErr)
	}

	sent := false
	badServe := acp.Provide(nil, badUTF8Runner{})[acp.OperationTask]
	if _, err := badServe(t.Context(), connection.Call{}, nil, func([]byte) error {
		sent = true
		return nil
	}); err == nil || sent {
		t.Errorf("badServe err = %v, sent = %v, want marshal error without calling send", err, sent)
	}
}

type badUTF8Runner struct{}

func (badUTF8Runner) RunTask(_ context.Context, _ connection.Call, _ *acp.TaskRequest, send func(*acp.TaskFrame) error) (*acp.TaskDone, error) {
	return nil, send(&acp.TaskFrame{
		Update: &acp.TaskFrame_AgentMessageChunk{AgentMessageChunk: &acp.MessageChunk{Text: "\xff"}},
	})
}

type protoMessage interface {
	proto.Message
	Reset()
	String() string
	Descriptor() ([]byte, []int)
}

func assertProtoEnum[E ~int32](
	t *testing.T, val E, enumFn func() *E, strFn func() string,
	descFn func() protoreflect.EnumDescriptor, typeFn func() protoreflect.EnumType,
	numFn func() protoreflect.EnumNumber, enumDescFn func() ([]byte, []int),
) {
	t.Helper()
	if got := enumFn(); got == nil || *got != val {
		t.Fatalf("Enum(%v) = %v", val, got)
	}
	if strFn() == "" {
		t.Fatalf("String(%v) is empty", val)
	}
	if descFn() == nil || typeFn() == nil || numFn() != protoreflect.EnumNumber(val) {
		t.Fatalf("reflect methods failed for %v", val)
	}
	if raw, idx := enumDescFn(); len(raw) == 0 || len(idx) == 0 {
		t.Fatalf("EnumDescriptor(%v) = len %d, %v", val, len(raw), idx)
	}
}

func assertProtoMessage(t *testing.T, msg protoMessage) {
	t.Helper()
	if msg.ProtoReflect() == nil || msg.ProtoReflect().Descriptor() == nil {
		t.Fatalf("ProtoReflect nil for %T", msg)
	}
	if raw, idx := msg.Descriptor(); len(raw) == 0 || len(idx) == 0 {
		t.Fatalf("Descriptor empty for %T", msg)
	}
	if msg.String() == "" {
		t.Fatalf("String empty for populated %T", msg)
	}
	msg.Reset()
	if msg.String() != "" {
		t.Fatalf("String non-empty after Reset for %T: %q", msg, msg.String())
	}
}

func TestGeneratedProtoMethods(t *testing.T) {
	for _, v := range []acp.PlanEntryPriority{
		acp.PlanEntryPriority_PLAN_ENTRY_PRIORITY_UNSPECIFIED,
		acp.PlanEntryPriority_PLAN_ENTRY_PRIORITY_HIGH,
		acp.PlanEntryPriority_PLAN_ENTRY_PRIORITY_MEDIUM,
		acp.PlanEntryPriority_PLAN_ENTRY_PRIORITY_LOW,
	} {
		assertProtoEnum(t, v, v.Enum, v.String, v.Descriptor, v.Type, v.Number, v.EnumDescriptor)
	}
	for _, v := range []acp.PlanEntryStatus{
		acp.PlanEntryStatus_PLAN_ENTRY_STATUS_UNSPECIFIED,
		acp.PlanEntryStatus_PLAN_ENTRY_STATUS_PENDING,
		acp.PlanEntryStatus_PLAN_ENTRY_STATUS_IN_PROGRESS,
		acp.PlanEntryStatus_PLAN_ENTRY_STATUS_COMPLETED,
	} {
		assertProtoEnum(t, v, v.Enum, v.String, v.Descriptor, v.Type, v.Number, v.EnumDescriptor)
	}
	for _, v := range []acp.ToolKind{
		acp.ToolKind_TOOL_KIND_UNSPECIFIED,
		acp.ToolKind_TOOL_KIND_READ,
		acp.ToolKind_TOOL_KIND_EDIT,
		acp.ToolKind_TOOL_KIND_DELETE,
		acp.ToolKind_TOOL_KIND_MOVE,
		acp.ToolKind_TOOL_KIND_SEARCH,
		acp.ToolKind_TOOL_KIND_EXECUTE,
		acp.ToolKind_TOOL_KIND_THINK,
		acp.ToolKind_TOOL_KIND_FETCH,
		acp.ToolKind_TOOL_KIND_OTHER,
	} {
		assertProtoEnum(t, v, v.Enum, v.String, v.Descriptor, v.Type, v.Number, v.EnumDescriptor)
	}
	for _, v := range []acp.ToolCallStatus{
		acp.ToolCallStatus_TOOL_CALL_STATUS_UNSPECIFIED,
		acp.ToolCallStatus_TOOL_CALL_STATUS_PENDING,
		acp.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS,
		acp.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED,
		acp.ToolCallStatus_TOOL_CALL_STATUS_FAILED,
	} {
		assertProtoEnum(t, v, v.Enum, v.String, v.Descriptor, v.Type, v.Number, v.EnumDescriptor)
	}
	for _, v := range []acp.StopReason{
		acp.StopReason_STOP_REASON_UNSPECIFIED,
		acp.StopReason_STOP_REASON_END_TURN,
		acp.StopReason_STOP_REASON_MAX_TOKENS,
		acp.StopReason_STOP_REASON_MAX_TURN_REQUESTS,
		acp.StopReason_STOP_REASON_REFUSAL,
		acp.StopReason_STOP_REASON_CANCELED,
	} {
		assertProtoEnum(t, v, v.Enum, v.String, v.Descriptor, v.Type, v.Number, v.EnumDescriptor)
	}

	title := "Title"
	kind := acp.ToolKind_TOOL_KIND_EDIT
	status := acp.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
	for _, msg := range []protoMessage{
		&acp.TaskRequest{Prompt: "p", VaultId: "v"},
		&acp.ContextNote{Path: "p", Title: "t", Text: "x"},
		&acp.TaskFrame{Update: &acp.TaskFrame_AgentMessageChunk{AgentMessageChunk: &acp.MessageChunk{Text: "hi"}}},
		&acp.MessageChunk{MessageId: "m", Text: "t"},
		&acp.Plan{Entries: []*acp.PlanEntry{{Content: "c"}}},
		&acp.PlanEntry{Content: "c", Priority: acp.PlanEntryPriority_PLAN_ENTRY_PRIORITY_HIGH, Status: acp.PlanEntryStatus_PLAN_ENTRY_STATUS_PENDING},
		&acp.ToolCallLocation{Path: "p", Line: 3},
		&acp.ToolCall{ToolCallId: "id", Title: "t", Kind: kind, Status: status, Content: []string{"c"}, Locations: []*acp.ToolCallLocation{{Path: "p"}}, RawInputJson: "{}"},
		&acp.ToolCallUpdate{ToolCallId: "id", Title: &title, Kind: &kind, Status: &status, Content: []string{"c"}, Locations: []*acp.ToolCallLocation{{Path: "p"}}, RawOutputJson: "{}"},
		&acp.Diff{Path: "p", OldText: "o", NewText: "n"},
		&acp.TaskDone{StopReason: acp.StopReason_STOP_REASON_END_TURN},
	} {
		assertProtoMessage(t, msg)
	}

	var nilReq *acp.TaskRequest
	var nilNote *acp.ContextNote
	var nilFrame *acp.TaskFrame
	var nilChunk *acp.MessageChunk
	var nilPlan *acp.Plan
	var nilEntry *acp.PlanEntry
	var nilLoc *acp.ToolCallLocation
	var nilCall *acp.ToolCall
	var nilUpdate *acp.ToolCallUpdate
	var nilDiff *acp.Diff
	var nilDone *acp.TaskDone
	for _, m := range []proto.Message{nilReq, nilNote, nilFrame, nilChunk, nilPlan, nilEntry, nilLoc, nilCall, nilUpdate, nilDiff, nilDone} {
		if m.ProtoReflect() == nil {
			t.Fatalf("nil ProtoReflect failed for %T", m)
		}
	}
}
