package manifest_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestTheHostRequiresItsExactPluginProtocol(t *testing.T) {
	for _, test := range []struct {
		plugin, host int
		want         bool
	}{
		{0, 1, true},
		{0, 2, false},
		{1, 1, true},
		{1, 2, false},
		{2, 1, false},
		{2, 2, true},
		{-1, 1, false},
	} {
		if got := manifest.IsProtocolCompatible(test.plugin, test.host); got != test.want {
			t.Errorf("plugin protocol %d on host %d: compatible = %v, want %v", test.plugin, test.host, got, test.want)
		}
	}
}
