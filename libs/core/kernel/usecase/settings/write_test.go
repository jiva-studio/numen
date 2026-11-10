package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func TestWritePassesTheSettingsOn(t *testing.T) {
	t.Parallel()
	var passed []settingsdomain.Setting
	write := settings.NewWrite(func(output []settingsdomain.Setting) error {
		passed = output
		return settingsdomain.ErrNotASetting
	})

	err := write.Execute([]settingsdomain.Setting{{Path: []string{"a"}, JSON: "1"}})

	if !errors.Is(err, settingsdomain.ErrNotASetting) || len(passed) != 1 {
		t.Fatalf("error is %v after passing %v", err, passed)
	}
}
