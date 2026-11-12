package model

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestCompleteAnswersThroughTheRowsOfThePluginTheRequestNames(t *testing.T) {
	handler, world, _ := newHandler(t)

	response, err := handler.Complete(t.Context(), connect.NewRequest(&v1.ModelServiceCompleteRequest{
		PluginId: viewPluginID, Messages: []*v1.ModelMessage{{Role: "user", Text: "fix"}},
	}))

	if err != nil || response.Msg.GetText() != "answer from "+address || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("Complete = %v, %v", response, err)
	}
	if got := world.Provider.Calls()[0].Request.Messages; len(got) != 1 || got[0].Text != "fix" {
		t.Errorf("messages = %v", got)
	}
}

func TestEmbedAnswersOneVectorForEachText(t *testing.T) {
	handler, _, _ := newHandler(t)

	response, err := handler.Embed(t.Context(), connect.NewRequest(&v1.ModelServiceEmbedRequest{
		PluginId: viewPluginID, Texts: []string{"ab", "abcd"}, Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))

	vectors := response.Msg.GetVectors()
	if err != nil || len(vectors) != 2 || vectors[1].GetValues()[0] != 4 {
		t.Errorf("Embed = %v, %v", response, err)
	}
}

func TestGetEmbeddingProfileAnswersTheProfileOfTheModelOfTheRow(t *testing.T) {
	handler, _, _ := newHandler(t)

	response, err := handler.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.ModelServiceGetEmbeddingProfileRequest{PluginId: viewPluginID}))

	msg := response.Msg
	if err != nil || msg.GetModel() != "model-1" || msg.GetDimensions() != 3 || msg.GetMaxTokens() != 8191 || msg.GetQueryPrefix() != "query: " {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestAPluginWithNoRowIsAServedNoConnection(t *testing.T) {
	handler, _, _ := newHandler(t)

	response, err := handler.Complete(t.Context(), connect.NewRequest(&v1.ModelServiceCompleteRequest{PluginId: "md.example.stranger"}))

	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Complete = %v, %v", response, err)
	}
}

func TestEveryRowThatFailedIsAServedAllFailedThatNamesEachConnection(t *testing.T) {
	handler, world, _ := newHandler(t)
	world.Provider.FailAlways(address, &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})

	response, err := handler.Embed(t.Context(), connect.NewRequest(&v1.ModelServiceEmbedRequest{PluginId: viewPluginID, Texts: []string{"a"}}))

	failures := response.Msg.GetFailures()
	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_ALL_FAILED ||
		len(failures) != 1 || failures[0].GetConnectionName() != "First" || failures[0].GetReason() != "payment required" {
		t.Errorf("Embed = %v, %v", response, err)
	}
}

func TestARejectedRequestIsAServedRejectedThatNamesTheConnection(t *testing.T) {
	handler, world, _ := newHandler(t)
	world.Provider.FailAlways(address, &connection.StopError{Reason: connection.FailureReasonBadRequest})

	response, err := handler.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.ModelServiceGetEmbeddingProfileRequest{PluginId: viewPluginID}))

	failures := response.Msg.GetFailures()
	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_REJECTED || len(failures) != 1 || failures[0].GetConnectionName() != "First" {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestARequestWithTheTokenOfAPluginIsRefusedByEveryCall(t *testing.T) {
	handler, world, entry := newHandler(t)
	complete := connect.NewRequest(&v1.ModelServiceCompleteRequest{PluginId: viewPluginID})
	embed := connect.NewRequest(&v1.ModelServiceEmbedRequest{PluginId: viewPluginID, Texts: []string{"a"}})
	profile := connect.NewRequest(&v1.ModelServiceGetEmbeddingProfileRequest{PluginId: viewPluginID})
	models := connect.NewRequest(&v1.ListModelsRequest{ConnectionId: entry.ID})
	levels := connect.NewRequest(&v1.ListReasoningLevelsRequest{ConnectionId: entry.ID})
	for _, request := range []connect.AnyRequest{complete, embed, profile, models, levels} {
		request.Header().Set("Authorization", "Bearer token-of-a-plugin")
	}

	_, completeErr := handler.Complete(t.Context(), complete)
	_, embedErr := handler.Embed(t.Context(), embed)
	_, profileErr := handler.GetEmbeddingProfile(t.Context(), profile)
	_, modelsErr := handler.ListModels(t.Context(), models)
	_, levelsErr := handler.ListReasoningLevels(t.Context(), levels)

	for name, err := range map[string]error{"Complete": completeErr, "Embed": embedErr, "GetEmbeddingProfile": profileErr, "ListModels": modelsErr, "ListReasoningLevels": levelsErr} {
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s with a plugin token: code = %v, want permission denied", name, connect.CodeOf(err))
		}
	}
	if got := world.Provider.Calls(); len(got) != 0 {
		t.Errorf("the provider received %d calls for refused requests", len(got))
	}
}

func TestEveryPriorityOfTheSchemaIsReadAsThePortOneIt(t *testing.T) {
	want := map[v1.Priority]modelcall.Priority{
		v1.Priority_PRIORITY_INTERACTIVE: modelcall.PriorityInteractive,
		v1.Priority_PRIORITY_BACKGROUND:  modelcall.PriorityBackground,
	}
	testsupport.CheckHandled(t, func(priority v1.Priority) bool { return readPriority(priority) == want[priority] })
}
