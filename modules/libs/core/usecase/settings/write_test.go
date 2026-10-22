package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestWritePassesTheSettingsOn(t *testing.T) {
	t.Parallel()
	var passed []port.Setting
	write := settings.NewWrite(func(output []port.Setting) error {
		passed = output
		return port.ErrNotASetting
	})

	err := write.Execute([]port.Setting{{Path: []string{"a"}, JSON: "1"}})

	if !errors.Is(err, port.ErrNotASetting) || len(passed) != 1 {
		t.Fatalf("error is %v after passing %v", err, passed)
	}
}
