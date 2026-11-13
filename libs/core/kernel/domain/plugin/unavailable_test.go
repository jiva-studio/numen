package plugin

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// An UnavailableError is ErrUnavailable regardless of its reason, keeps its
// cause, and reports which plugin and why.
func TestUnavailableErrorIsErrUnavailable(t *testing.T) {
	cause := errors.New("exit status 1")
	for _, tc := range []struct {
		err  *UnavailableError
		want string
	}{
		{&UnavailableError{Capability: "book", Reason: NotInstalled}, "the plugin for book is not installed"},
		{&UnavailableError{Capability: "book", Reason: Disabled}, "the plugin for book is disabled"},
		{&UnavailableError{PluginID: "md.numen.readers.epub", Reason: Disabled}, "plugin md.numen.readers.epub is disabled"},
		{&UnavailableError{PluginID: "md.numen.readers.epub", Reason: Failed, Err: cause}, "plugin md.numen.readers.epub failed to start: exit status 1"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
		if !errors.Is(tc.err, domain.ErrUnavailable) {
			t.Errorf("%q is not ErrUnavailable", tc.want)
		}
	}
	if !errors.Is(&UnavailableError{Reason: Failed, Err: cause}, cause) {
		t.Error("the cause is lost")
	}
}
