package platform_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/platform"
)

func TestIsPhoneSystem_IsTrueForAndroidAndIos(t *testing.T) {
	for goos, want := range map[string]bool{"android": true, "ios": true, "linux": false, "darwin": false, "windows": false} {
		if got := platform.IsPhoneSystem(goos); got != want {
			t.Errorf("%s: got %v, want %v", goos, got, want)
		}
	}
}
