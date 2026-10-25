package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/settings"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

func TestWriteFilePassesTheTextAndTheBaselineOn(t *testing.T) {
	t.Parallel()
	var text string
	var from *string
	write := settings.NewWriteFile(func(output string, baseline *string) error {
		text, from = output, baseline
		return vaultport.ErrStale
	})
	baseline := "was"

	err := write.Execute("is", &baseline)

	if !errors.Is(err, vaultport.ErrStale) || text != "is" || from == nil || *from != "was" {
		t.Fatalf("error is %v after passing %q over %v", err, text, from)
	}
}
