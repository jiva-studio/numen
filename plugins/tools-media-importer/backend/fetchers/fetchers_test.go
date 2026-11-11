package fetchers

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/tempfiles"
)

const videoAddress = "https://www.youtube.com/watch?v=tidalmill01"

// The settings name the programs, the captions and how large a copy may be.
func TestUpdateConfigSetsToolsCaptionsAndTheLargestCopy(t *testing.T) {
	fMock := New(true, process.Runner{}, tempfiles.Files{})
	if fMock.GetFetcher() == nil || fMock.GetFetcher().GetDownloadModel(videoAddress).Tool != "mock" {
		t.Fatalf("mock fetcher = %+v", fMock.GetFetcher())
	}
	f := New(false, process.Runner{}, tempfiles.Files{})
	if f.GetFetcher() == nil || f.GetFetcher().GetDownloadModel(videoAddress).Tool == "mock" {
		t.Fatalf("real fetcher = %+v", f.GetFetcher())
	}
	if f.GetSettings().CopyMaxSize != 500<<20 {
		t.Errorf("default CopyMaxSize = %d, want %d", f.GetSettings().CopyMaxSize, 500<<20)
	}
	before := f.GetFetcher()
	f.UpdateConfig(map[string]string{
		"ytDlp": "/custom/yt-dlp", "ffmpeg": "/custom/ffmpeg",
		"captions": " sa, en ,", "automaticCaptions": "false", "copyMaxSizeMb": "20",
	})
	if f.GetFetcher() == before || f.GetFetcher() == nil || f.GetFetcher().GetDownloadModel(videoAddress).Tool != "yt-dlp" {
		t.Error("updateConfig did not reopen current fetcher")
	}
	if !slices.Equal(f.GetPrograms().Video.Command, []string{"/custom/yt-dlp"}) || !slices.Equal(f.GetPrograms().Sound.Command, []string{"/custom/ffmpeg"}) {
		t.Errorf("the programs are %v and %v", f.GetPrograms().Video.Command, f.GetPrograms().Sound.Command)
	}
	settings := f.GetSettings()
	if !slices.Equal(settings.Captions, []string{"sa", "en"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("the settings are %+v", settings)
	}
	f.UpdateConfig(map[string]string{"ytDlp": "", "ffmpeg": "", "copyMaxSizeMb": "many"})
	if len(f.GetPrograms().Video.Command) != 0 || len(f.GetPrograms().Sound.Command) != 0 || f.GetSettings().CopyMaxSize != 20<<20 {
		t.Errorf("cleared settings leave %v, %v and %d", f.GetPrograms().Video.Command, f.GetPrograms().Sound.Command, f.GetSettings().CopyMaxSize)
	}
	f.UpdateConfig(map[string]string{"copyMaxSizeMb": "-1"})
	if f.GetSettings().CopyMaxSize != 20<<20 {
		t.Errorf("negative size changed CopyMaxSize to %d", f.GetSettings().CopyMaxSize)
	}
	f.UpdateConfig(map[string]string{"copyMaxSizeMb": "0"})
	if f.GetSettings().CopyMaxSize != 0 {
		t.Errorf("zero size left CopyMaxSize = %d, want 0", f.GetSettings().CopyMaxSize)
	}
	f.UpdateConfig(map[string]string{"copyMaxSizeMb": "20"})
	f.UpdateConfig(map[string]string{"copyMaxSizeMb": "4611686018427387904"})
	if f.GetSettings().CopyMaxSize != 0 {
		t.Errorf("64-bit size parse failed: CopyMaxSize = %d, want 0", f.GetSettings().CopyMaxSize)
	}
}

func TestImportSettingsAreReadFromTheCamelCaseKeys(t *testing.T) {
	f := New(false, process.Runner{}, tempfiles.Files{})
	f.UpdateConfig(map[string]string{
		"ytDlp": "/custom/yt-dlp", "automaticCaptions": "false", "copyMaxSizeMb": "20",
	})

	settings := f.GetSettings()
	if !slices.Equal(f.GetPrograms().Video.Command, []string{"/custom/yt-dlp"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("video program %v, automatic captions %t, largest copy %d; want /custom/yt-dlp, false and %d",
			f.GetPrograms().Video.Command, settings.IsAutomatic, settings.CopyMaxSize, int64(20<<20))
	}
}

func TestImportSettingsAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	f := New(false, process.Runner{}, tempfiles.Files{})
	before, command := f.GetSettings(), slices.Clone(f.GetPrograms().Video.Command)
	f.UpdateConfig(map[string]string{
		"yt_dlp": "/custom/yt-dlp", "automatic_captions": "false", "copy_max_size_mb": "20",
	})

	after := f.GetSettings()
	if after.IsAutomatic != before.IsAutomatic || after.CopyMaxSize != before.CopyMaxSize || !slices.Equal(f.GetPrograms().Video.Command, command) {
		t.Errorf("automatic captions %t, largest copy %d and video program %v, want them unchanged at %t, %d and %v",
			after.IsAutomatic, after.CopyMaxSize, f.GetPrograms().Video.Command, before.IsAutomatic, before.CopyMaxSize, command)
	}
}
