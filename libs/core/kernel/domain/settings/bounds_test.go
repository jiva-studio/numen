package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func TestANumberInsideTheBoundsIsAccepted(t *testing.T) {
	bounds := settings.Bounds{Min: 0.8, Max: 1.75}
	for _, number := range []float64{0.8, 1, 1.75} {
		if err := bounds.Check("app.appearance.textScale", number); err != nil {
			t.Errorf("%v was refused: %v", number, err)
		}
	}
}

func TestANumberOutsideTheBoundsNamesTheSettingAndTheRange(t *testing.T) {
	bounds := settings.Bounds{Min: 0.8, Max: 1.75}
	for _, number := range []float64{0.79, 9} {
		err := bounds.Check("-text-scale", number)
		var outside *settings.OutsideBoundsError
		if !errors.As(err, &outside) || outside.At != "-text-scale" || outside.Number != number || outside.Bounds != bounds {
			t.Fatalf("%v gave %#v", number, err)
		}
	}
	if want := `"-text-scale": 9 is out of range 0.8..1.75`; bounds.Check("-text-scale", 9).Error() != want {
		t.Errorf("said %v, want %q", bounds.Check("-text-scale", 9), want)
	}
}
