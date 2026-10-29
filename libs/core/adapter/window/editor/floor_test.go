package editor

import (
	"errors"
	"os"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/container"
)

// A window opened with no model answers the floor requests it can: the floor is
// the default, and what needs a model is unavailable.
func TestAWindowWithNoModelAnswersFloorRequests(t *testing.T) {
	installation, err := Open(
		t.Context(), NewAssembly(t, vaultConfig(t, container.EmbeddingConfig{})), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installation.Close() })
	search := installation.Services.Search

	got, err := search.GetFloor(t.Context(), connect.NewRequest(&v1.GetFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetSource() != v1.FloorSource_FLOOR_SOURCE_DEFAULT || got.Msg.GetFloor() != 0.30 || got.Msg.GetModel() != "" {
		t.Errorf("got %v", got.Msg)
	}

	_, err = search.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{}))
	var failure *connect.Error
	if err == nil || !errors.As(err, &failure) || failure.Code() != connect.CodeUnavailable {
		t.Errorf("calibrating with no model gave %v, want Unavailable", err)
	}
}

// A settings file that names a floor is the floor in force, whatever the vault holds.
func TestTheFloorOfTheSettingsFileIsInForceInAWindow(t *testing.T) {
	installation, err := Open(
		t.Context(), NewAssembly(t, vaultConfig(t, container.EmbeddingConfig{Floor: 0.58})), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installation.Close() })

	got, err := installation.Services.Search.GetFloor(t.Context(), connect.NewRequest(&v1.GetFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetSource() != v1.FloorSource_FLOOR_SOURCE_FILE || got.Msg.GetFloor() != 0.58 {
		t.Errorf("got %v", got.Msg)
	}
}
