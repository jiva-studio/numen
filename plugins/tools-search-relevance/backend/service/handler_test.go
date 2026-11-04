package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
)

func TestHandlersRefuseCallsBeforeTheAppIsMounted(t *testing.T) {
	app := NewApp()
	if _, err := app.getFloor(t.Context(), api.GetFloorRequest{}); !errors.Is(err, errNotMounted) {
		t.Errorf("getFloor before mount gave %v, want errNotMounted", err)
	}
	if _, err := app.calibrateFloor(t.Context(), api.CalibrateFloorRequest{}); !errors.Is(err, errNotMounted) {
		t.Errorf("calibrateFloor before mount gave %v, want errNotMounted", err)
	}
	if _, err := app.setFloor(t.Context(), api.SetFloorRequest{}); !errors.Is(err, errNotMounted) {
		t.Errorf("setFloor before mount gave %v, want errNotMounted", err)
	}
	if _, err := app.probeFloor(t.Context(), api.ProbeFloorRequest{}); !errors.Is(err, errNotMounted) {
		t.Errorf("probeFloor before mount gave %v, want errNotMounted", err)
	}
}
