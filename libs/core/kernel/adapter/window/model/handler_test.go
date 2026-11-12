package model

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	modelcalls "github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modellist"
)

const (
	address      = "https://one.test/v1"
	viewPluginID = "md.example.view"
)

func newHandler(t *testing.T) (*Handler, *modelcalltest.World, modelcalltest.Entry) {
	t.Helper()
	world := modelcalltest.New()
	entry := world.AddServer(t, "First", address, "sk-one")
	world.Rows.Set(viewPluginID, "complete", modelcalltest.Row(entry, "model-1"))
	world.Rows.Set(viewPluginID, "embed", modelcalltest.Row(entry, "model-1"))
	calls := modelcalls.NewCalls(world.Resolve, world.ReadSecrets, world.Rows, world.Source, world.Clock, world.Waits.Wait, world.Notices)
	return NewHandler(modellist.NewLists(world.Resolve, world.ReadSecrets, world.Source), calls), world, entry
}

func TestListModelsReturnsTheModelsWithTheirLabels(t *testing.T) {
	handler, _, entry := newHandler(t)

	response, err := handler.ListModels(t.Context(), connect.NewRequest(&v1.ListModelsRequest{ConnectionId: entry.ID, Operation: "embed"}))

	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("ListModels = %v, %v", response, err)
	}
	models := response.Msg.GetModels()
	if len(models) != 2 || models[0].GetId() != "model-1" || models[0].GetTitle() != "Model one" ||
		len(models[0].GetOperations()) != 1 || models[0].GetOperations()[0] != "embed" || len(models[1].GetOperations()) != 0 {
		t.Errorf("models = %v", models)
	}
}

func TestListReasoningLevelsReturnsTheLevelsOfTheModel(t *testing.T) {
	handler, world, entry := newHandler(t)

	response, err := handler.ListReasoningLevels(t.Context(), connect.NewRequest(&v1.ListReasoningLevelsRequest{ConnectionId: entry.ID, Model: "model-1"}))

	levels := response.Msg.GetLevels()
	if err != nil || len(levels) != 2 || levels[0].GetValue() != "low" || levels[0].GetTitle() != "Low" {
		t.Errorf("ListReasoningLevels = %v, %v", response, err)
	}
	if got := world.Provider.Calls()[0].Call.Model; got != "model-1" {
		t.Errorf("the provider was asked about model %q", got)
	}
}

func TestAConnectionThatIsNotInTheListIsAServedNotFound(t *testing.T) {
	handler, _, _ := newHandler(t)

	response, err := handler.ListModels(t.Context(), connect.NewRequest(&v1.ListModelsRequest{ConnectionId: "c0000dead"}))

	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_NOT_FOUND {
		t.Errorf("ListModels = %v, %v", response, err)
	}
}

func TestAProviderThatFailedIsAServedProviderUnavailableWithTheReason(t *testing.T) {
	handler, world, entry := newHandler(t)
	world.Provider.FailAlways(address, &connection.TryNextError{Reason: connection.FailureReasonKeyRejected})

	response, err := handler.ListReasoningLevels(t.Context(), connect.NewRequest(&v1.ListReasoningLevelsRequest{ConnectionId: entry.ID}))

	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_PROVIDER_UNAVAILABLE || response.Msg.GetDetail() != "key rejected" {
		t.Errorf("ListReasoningLevels = %v, %v", response, err)
	}
}

func TestAConnectionWhoseTypeIsNotProvidedIsAServedProviderUnavailable(t *testing.T) {
	handler, world, entry := newHandler(t)
	world.SetAvailable(modelcalltest.ProviderPluginID, false)

	response, err := handler.ListModels(t.Context(), connect.NewRequest(&v1.ListModelsRequest{ConnectionId: entry.ID}))

	if err != nil || response.Msg.GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_PROVIDER_UNAVAILABLE || response.Msg.GetDetail() != "type unavailable" {
		t.Errorf("ListModels = %v, %v", response, err)
	}
}

func TestACancelledAskIsAStatusCode(t *testing.T) {
	handler, _, entry := newHandler(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := handler.ListModels(ctx, connect.NewRequest(&v1.ListModelsRequest{ConnectionId: entry.ID}))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled", connect.CodeOf(err))
	}
}

func TestAnyOtherFailureIsAStatusCode(t *testing.T) {
	handler, world, entry := newHandler(t)
	world.Keys.OnGet = func() { world.ChangeRevision(t, entry.ID) }

	_, err := handler.ListModels(t.Context(), connect.NewRequest(&v1.ListModelsRequest{ConnectionId: entry.ID}))

	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want internal: %v", connect.CodeOf(err), err)
	}
}

func TestErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.ModelErrorCode]error{
		v1.ModelErrorCode_MODEL_ERROR_CODE_NOT_FOUND:            connection.ErrNotFound,
		v1.ModelErrorCode_MODEL_ERROR_CODE_PROVIDER_UNAVAILABLE: modellist.ErrProviderUnavailable,
		v1.ModelErrorCode_MODEL_ERROR_CODE_NO_CONNECTION:        connection.ErrNoConnection,
		v1.ModelErrorCode_MODEL_ERROR_CODE_ALL_FAILED:           &connection.AllFailedError{},
		v1.ModelErrorCode_MODEL_ERROR_CODE_REJECTED:             &connection.RejectedError{},
	}, func(failure error) v1.ModelErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
	if _, isServed := getErrorCode(errors.New("other")); isServed {
		t.Error("an error of no kind is served")
	}
}
