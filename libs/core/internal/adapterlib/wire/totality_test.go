package wire

import (
	"errors"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Every value of every enum this file converts maps to a defined result.
//
// The values are enumerated from the schema, so a value added there fails this
// test and is not silently ignored.

func TestEveryModeCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, ConvertModeFromWire, ConvertModeToWire)
}

func TestEveryUnitIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Unit]run.Unit{
		v1.Unit_UNIT_THINGS:  run.Things,
		v1.Unit_UNIT_BYTES:   run.Bytes,
		v1.Unit_UNIT_SECONDS: run.Seconds,
	}, convertUnit)
}

// A page that reported nothing is unanswered, and the round waits its bound for
// unanswered pages. Every answer the schema defines counts as an answer.
func TestEveryFlushResultIsAnAnswer(t *testing.T) {
	testsupport.CheckHandled(t, func(response v1.FlushResult) bool { return parseFlushResult(response) != flushUnanswered })
}

func TestEveryRoleCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, func(role v1.Role) domain.LinkRole {
		read, _ := ConvertRoleFromWire(role)
		return read
	}, ConvertRoleToWire)
	testsupport.CheckHandled(t, func(role v1.Role) bool {
		_, named := ConvertRoleFromWire(role)
		return named
	})
}

func TestEveryMarkerCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, func(marker v1.Marker) domain.LinkMarker {
		read, _ := ConvertMarkerFromWire(marker)
		return read
	}, ConvertMarkerToWire)
	testsupport.CheckHandled(t, func(marker v1.Marker) bool {
		_, named := ConvertMarkerFromWire(marker)
		return named
	})
}

func TestAConnectionCrossesTheWireWithBothSides(t *testing.T) {
	c := domain.Connection{
		A: "a.md", B: "b.md",
		AB: &domain.Side{Role: domain.RoleParent, Type: "requires", Why: "why", Label: "l", Marker: domain.MarkerArrow},
	}

	back, err := ConvertConnectionFromWire(ConvertConnectionToWire(c))

	if err != nil {
		t.Fatal(err)
	}
	if back.A != c.A || back.B != c.B || back.BA != nil || back.AB == nil || *back.AB != *c.AB {
		t.Errorf("back = %+v", back)
	}
}

func TestASideNamingNoRoleIsRefused(t *testing.T) {
	_, err := ConvertConnectionFromWire(&v1.Connection{
		First: "a.md", Second: "b.md", FirstSide: &v1.ConnectionSide{},
	})
	if !errors.Is(err, ErrBadWireValue) {
		t.Errorf("err = %v, want ErrBadWireValue", err)
	}
}
