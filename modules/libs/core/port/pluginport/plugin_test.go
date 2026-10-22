package pluginport

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
)

// An UnavailableError is ErrUnavailable whatever its reason, keeps its cause,
// and says which plugin and why.
func TestUnavailableErrorIsErrUnavailable(t *testing.T) {
	cause := errors.New("exit status 1")
	for _, tc := range []struct {
		err  *UnavailableError
		want string
	}{
		{&UnavailableError{Capability: "book", Reason: PluginNotInstalled}, "the plugin for book is not installed"},
		{&UnavailableError{Capability: "book", Reason: PluginDisabled}, "the plugin for book is disabled"},
		{&UnavailableError{PluginID: "md.numen.readers.epub", Reason: PluginDisabled}, "plugin md.numen.readers.epub is disabled"},
		{&UnavailableError{PluginID: "md.numen.readers.epub", Reason: PluginFailed, Err: cause}, "plugin md.numen.readers.epub failed to start: exit status 1"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
		if !errors.Is(tc.err, completionport.ErrUnavailable) {
			t.Errorf("%q is not ErrUnavailable", tc.want)
		}
		if errors.Is(tc.err, completionport.ErrNoRoute) {
			t.Errorf("%q is ErrNoRoute", tc.want)
		}
	}
	if !errors.Is(&UnavailableError{Reason: PluginFailed, Err: cause}, cause) {
		t.Error("the cause is lost")
	}
}
