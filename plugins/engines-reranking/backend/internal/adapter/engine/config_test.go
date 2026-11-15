package engine

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/onnx"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for _, one := range []struct {
		name string
		want onnx.Provider
	}{{"", onnx.ProviderCPU}, {"auto", onnx.ProviderCPU}, {" AUTO ", onnx.ProviderCPU}, {"cuda", onnx.ProviderCUDA}} {
		t.Setenv("NUMEN_ONNX_PROVIDER", one.name)
		if got := getSessionProvider(); got != one.want {
			t.Errorf("with %q, getSessionProvider() = %q, want %q", one.name, got, one.want)
		}
	}
}
