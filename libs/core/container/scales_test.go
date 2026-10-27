package container_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
)

func TestCheckInterfaceScaleRejectsAScaleOutsideItsBounds(t *testing.T) {
	if err := container.CheckInterfaceScale("-interface-scale", 1.25); err != nil {
		t.Errorf("CheckInterfaceScale(1.25) = %v, want nil", err)
	}
	err := container.CheckInterfaceScale("-interface-scale", 9)
	if err == nil || !strings.Contains(err.Error(), "-interface-scale") {
		t.Errorf("CheckInterfaceScale(9) = %v, want an error naming the flag", err)
	}
}

func TestCheckTextScaleRejectsAScaleOutsideItsBounds(t *testing.T) {
	if err := container.CheckTextScale("-text-scale", 1); err != nil {
		t.Errorf("CheckTextScale(1) = %v, want nil", err)
	}
	err := container.CheckTextScale("-text-scale", 0.1)
	if err == nil || !strings.Contains(err.Error(), "-text-scale") {
		t.Errorf("CheckTextScale(0.1) = %v, want an error naming the flag", err)
	}
}
