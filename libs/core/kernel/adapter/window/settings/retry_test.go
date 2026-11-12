package settings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func newRetryHandler(t *testing.T, retry func(context.Context, string) (*plugin.Failure, error)) *Handler {
	t.Helper()
	useCase := setting.NewRetryPlugin(retry)
	return NewHandler(Values{}, NewPluginFolders(useCase, nil, nil))
}

// A plugin that starts again is answered with no failure.
func TestRetryPlugin_SuccessReturnsNoFailure(t *testing.T) {
	var requestedID string
	handler := newRetryHandler(t, func(_ context.Context, id string) (*plugin.Failure, error) {
		requestedID = id
		return nil, nil
	})

	answer, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || answer.Msg.GetFailure() != "" || requestedID != "md.numen.sample" {
		t.Fatalf("answered %v, %v for %q", answer, err, requestedID)
	}
}

// A plugin that fails again is answered with its reason.
func TestRetryPlugin_FailureReturnsReason(t *testing.T) {
	handler := newRetryHandler(t, func(context.Context, string) (*plugin.Failure, error) {
		return &plugin.Failure{PluginID: "md.numen.sample", Reason: "Timed out while starting"}, nil
	})

	answer, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || answer.Msg.GetFailure() != "Timed out while starting" {
		t.Fatalf("answered %v, %v", answer, err)
	}
}

func TestARetryOfAPluginThatIsNotInstalledIsNotFound(t *testing.T) {
	missing := newRetryHandler(t, func(_ context.Context, id string) (*plugin.Failure, error) {
		return nil, &plugin.UnavailableError{PluginID: id, Reason: plugin.NotInstalled}
	})
	broken := newRetryHandler(t, func(context.Context, string) (*plugin.Failure, error) {
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

// A build that serves no settings answers that it serves none.
func TestABuildWithoutSettingsServesNoRetry(t *testing.T) {
	_, err := (&Handler{}).RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("retry answered %v", err)
	}
}
