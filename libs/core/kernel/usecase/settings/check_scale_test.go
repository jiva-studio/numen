package settings_test

import (
	"errors"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func TestAScaleInsideTheRangeIsAccepted(t *testing.T) {
	check := settingsusecase.NewCheckScale(settingsdomain.Bounds{Min: 0.8, Max: 2})
	if err := check.Execute("-interface-scale", 1.25); err != nil {
		t.Errorf("1.25 was refused: %v", err)
	}
}

func TestAScaleOutsideTheRangeIsRefusedWithTheNameGiven(t *testing.T) {
	check := settingsusecase.NewCheckScale(settingsdomain.Bounds{Min: 0.8, Max: 2})
	err := check.Execute("-interface-scale", 9)
	var outside *settingsdomain.OutsideBoundsError
	if !errors.As(err, &outside) || outside.At != "-interface-scale" || outside.Number != 9 {
		t.Fatalf("said %#v", err)
	}
}
