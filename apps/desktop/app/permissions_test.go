package app

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

func TestTheMainWindowAllowsTheCameraAndTheMicrophone(t *testing.T) {
	opened, _ := windowOn(t)

	got := getMainWindowOptions(opened).Permissions

	if got[application.PermissionCamera] != application.PermissionAllow || got[application.PermissionMicrophone] != application.PermissionAllow {
		t.Errorf("permissions = %v, want the camera and the microphone allowed", got)
	}
}

func TestASecondaryWindowDeniesTheCameraAndTheMicrophone(t *testing.T) {
	opened, _ := windowOn(t)

	got := getSecondaryWindowOptions(opened, windowdomain.DetachedTab{ID: "tab"}).Permissions

	if got[application.PermissionCamera] != application.PermissionDeny || got[application.PermissionMicrophone] != application.PermissionDeny {
		t.Errorf("permissions = %v, want both denied", got)
	}
}
