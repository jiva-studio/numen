package settings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

func newStopHandler(
	t *testing.T, stop func(context.Context, string, string) (*setting.PluginFailure, error),
) *Handler {
	t.Helper()
	useCase, err := setting.NewStopFailedPlugin(stop)
	if err != nil {
		t.Fatal(err)
	}
	return &Handler{PluginStop: useCase}
}

func TestAStopOfAFailedPluginPassesTheIdAndTheReason(t *testing.T) {
	var gotID, gotReason string
	handler := newStopHandler(t, func(_ context.Context, id, reason string) (*setting.PluginFailure, error) {
		gotID, gotReason = id, reason
		return &setting.PluginFailure{PluginID: id}, nil
	})

	_, err := handler.StopFailedPlugin(t.Context(), connect.NewRequest(&v1.StopFailedPluginRequest{
		PluginId: "md.numen.sample", Reason: "the plugin did not start within 30 s",
	}))

	if err != nil || gotID != "md.numen.sample" || gotReason != "the plugin did not start within 30 s" {
		t.Fatalf("answered %v for %q, %q", err, gotID, gotReason)
	}
}

func TestAStopOfAPluginThatIsNotInstalledIsNotFound(t *testing.T) {
	missing := newStopHandler(t, func(_ context.Context, id, _ string) (*setting.PluginFailure, error) {
		return nil, &setting.UnavailableError{PluginID: id, Reason: setting.PluginNotInstalled}
	})
	broken := newStopHandler(t, func(context.Context, string, string) (*setting.PluginFailure, error) {
		return nil, errors.New("the host is closed")
	})
	request := connect.NewRequest(&v1.StopFailedPluginRequest{PluginId: "md.numen.missing"})

	_, notFound := missing.StopFailedPlugin(t.Context(), request)
	_, internal := broken.StopFailedPlugin(t.Context(), request)
	_, unimplemented := (&Handler{}).StopFailedPlugin(t.Context(), request)

	if connect.CodeOf(notFound) != connect.CodeNotFound || connect.CodeOf(internal) != connect.CodeInternal ||
		connect.CodeOf(unimplemented) != connect.CodeUnimplemented {
		t.Fatalf("codes = %v, %v, %v", connect.CodeOf(notFound), connect.CodeOf(internal), connect.CodeOf(unimplemented))
	}
}
