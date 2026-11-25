package plugin

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func newRetryHandler(t *testing.T, retry func(context.Context, string) (*plugindomain.Failure, error)) *Handler {
	t.Helper()
	useCase := setting.NewRetryPlugin(retry)
	return NewHandler(nil, useCase)
}

// A plugin that starts again is answered with no failure.
func TestRetryPlugin_SuccessReturnsNoFailure(t *testing.T) {
	var requestedID string
	handler := newRetryHandler(t, func(_ context.Context, id string) (*plugindomain.Failure, error) {
		requestedID = id
		return nil, nil
	})

	answer, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED || answer.Msg.GetDetail() != "" || requestedID != "md.numen.sample" {
		t.Fatalf("answered %v, %v for %q", answer, err, requestedID)
	}
}

// A plugin that fails again is answered with its reason.
func TestRetryPlugin_FailureReturnsReason(t *testing.T) {
	handler := newRetryHandler(t, func(context.Context, string) (*plugindomain.Failure, error) {
		return &plugindomain.Failure{PluginID: "md.numen.sample", Reason: "Timed out while starting"}, nil
	})

	answer, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_INTERNAL || answer.Msg.GetDetail() != "Timed out while starting" {
		t.Fatalf("answered %v, %v", answer, err)
	}
}

func TestARetryOfAPluginThatIsNotInstalledIsNotFound(t *testing.T) {
	missing := newRetryHandler(t, func(_ context.Context, id string) (*plugindomain.Failure, error) {
		return nil, &plugindomain.UnavailableError{PluginID: id, Reason: plugindomain.NotInstalled}
	})
	broken := newRetryHandler(t, func(context.Context, string) (*plugindomain.Failure, error) {
		return nil, errors.New("the host is closed")
	})

	_, err := missing.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a missing plugin answered %v", err)
	}
	_, err = broken.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("a closed host answered %v", err)
	}
}

// A build that cannot start a plugin again answers that it cannot.
func TestABuildWithoutRetryServesNoRetry(t *testing.T) {
	_, err := (&Handler{}).RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("retry answered %v", err)
	}
}
