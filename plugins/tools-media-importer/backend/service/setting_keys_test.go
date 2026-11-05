package service

import (
	"slices"
	"testing"
)

func TestImportSettingsAreReadFromTheCamelCaseKeys(t *testing.T) {
	f := newFetchers(false)
	f.updateConfig(map[string]string{
		"ytDlp": "/custom/yt-dlp", "automaticCaptions": "false", "copyMaxSizeMb": "20",
	})

	settings := f.getSettings()
	if !slices.Equal(f.cfg.Video.Command, []string{"/custom/yt-dlp"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("video program %v, automatic captions %t, largest copy %d; want /custom/yt-dlp, false and %d",
			f.cfg.Video.Command, settings.IsAutomatic, settings.CopyMaxSize, int64(20<<20))
	}
}

func TestImportSettingsAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	f := newFetchers(false)
	before, command := f.getSettings(), slices.Clone(f.cfg.Video.Command)
	f.updateConfig(map[string]string{
		"yt_dlp": "/custom/yt-dlp", "automatic_captions": "false", "copy_max_size_mb": "20",
	})

	after := f.getSettings()
	if after.IsAutomatic != before.IsAutomatic || after.CopyMaxSize != before.CopyMaxSize || !slices.Equal(f.cfg.Video.Command, command) {
		t.Errorf("automatic captions %t, largest copy %d and video program %v, want them unchanged at %t, %d and %v",
			after.IsAutomatic, after.CopyMaxSize, f.cfg.Video.Command, before.IsAutomatic, before.CopyMaxSize, command)
	}
}
