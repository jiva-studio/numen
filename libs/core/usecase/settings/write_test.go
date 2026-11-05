package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/settings"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

func TestWritePassesTheSettingsOn(t *testing.T) {
	t.Parallel()
	var passed []settingsport.Setting
	write := settings.NewWrite(func(output []settingsport.Setting) error {
		passed = output
		return settingsport.ErrNotASetting
	})

	err := write.Execute([]settingsport.Setting{{Path: []string{"a"}, JSON: "1"}})

	if !errors.Is(err, settingsport.ErrNotASetting) || len(passed) != 1 {
		t.Fatalf("error is %v after passing %v", err, passed)
	}
}
