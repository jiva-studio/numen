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
	isApplied := false
	write := settings.NewWrite(func(output []settingsport.Setting, applied func()) error {
		passed = output
		applied()
		return settingsport.ErrNotASetting
	})

	err := write.Execute([]settingsport.Setting{{Path: []string{"a"}, JSON: "1"}}, func() { isApplied = true })

	if !errors.Is(err, settingsport.ErrNotASetting) || len(passed) != 1 || !isApplied {
		t.Fatalf("error is %v after passing %v, applied %v", err, passed, isApplied)
	}
}
