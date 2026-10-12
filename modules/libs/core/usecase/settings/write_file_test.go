package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestWriteFilePassesTheTextAndTheBaselineOn(t *testing.T) {
	t.Parallel()
	var text string
	var from *string
	write := settings.NewWriteFile(func(written string, baseline *string) error {
		text, from = written, baseline
		return port.ErrStale
	})
	baseline := "was"

	err := write.Execute("is", &baseline)

	if !errors.Is(err, port.ErrStale) || text != "is" || from == nil || *from != "was" {
		t.Fatalf("error is %v after passing %q over %v", err, text, from)
	}
}
