package editor

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	setting "github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func newRetryHandler(retry func(context.Context, string) (*setting.PluginFailure, error)) *settingsHandler {
	return &settingsHandler{retryPlugin: setting.NewRetryPlugin(retry)}
}

// A plugin that starts again is answered with no failure.
func TestARetriedPluginThatStartsIsAnsweredWithNoFailure(t *testing.T) {
	var asked string
	handler := newRetryHandler(func(_ context.Context, id string) (*setting.PluginFailure, error) {
		asked = id
		return nil, nil
	})

	said, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || said.Msg.GetFailure() != "" || asked != "md.numen.sample" {
		t.Fatalf("answered %v, %v for %q", said, err, asked)
	}
}

// A plugin that fails again is answered with its reason.
func TestARetriedPluginThatFailsAgainIsAnsweredWithItsReason(t *testing.T) {
	handler := newRetryHandler(func(context.Context, string) (*setting.PluginFailure, error) {
		return &setting.PluginFailure{PluginID: "md.numen.sample", Reason: "The plugin did not finish starting in time."}, nil
	})

	said, err := handler.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.sample"}))

	if err != nil || said.Msg.GetFailure() != "The plugin did not finish starting in time." {
		t.Fatalf("answered %v, %v", said, err)
	}
}

// A plugin nobody installed is not found, and any other refusal is the core's.
func TestARetryOfAPluginThatIsNotInstalledIsNotFound(t *testing.T) {
	missing := newRetryHandler(func(_ context.Context, id string) (*setting.PluginFailure, error) {
		return nil, &setting.UnavailableError{PluginID: id, Reason: setting.PluginNotInstalled}
	})
	broken := newRetryHandler(func(context.Context, string) (*setting.PluginFailure, error) {
		return nil, errors.New("the host is closed")
	})

	_, err := missing.RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{PluginId: "md.numen.none"}))
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
	_, err := (&settingsHandler{}).RetryPlugin(t.Context(), connect.NewRequest(&v1.RetryPluginRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("retry answered %v", err)
	}
}

// The failure of a plugin travels to the window with its reason and its time, and a plugin that has not failed travels with none.
func TestAFailedPluginTravelsWithItsReasonAndTime(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_123)
	infos := newWirePluginInfos([]setting.PluginInfo{
		{ID: "md.numen.sample", Failure: &setting.PluginFailure{Reason: "The plugin did not finish starting in time.", At: at}},
		{ID: "md.numen.other"},
	})

	failure := infos[0].GetFailure()
	if failure.GetReason() != "The plugin did not finish starting in time." || failure.GetFailedAtUnixMs() != at.UnixMilli() {
		t.Errorf("travelled as %v", failure)
	}
	if infos[1].GetFailure() != nil {
		t.Errorf("a plugin that has not failed travelled as %v", infos[1].GetFailure())
	}
}
