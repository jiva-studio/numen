package fetchers

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/tempfiles"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
)

const videoAddress = "https://www.youtube.com/watch?v=tidalmill01"

// The settings name the programs, the captions and how large a copy may be.
func TestUpdateConfigSetsToolsCaptionsAndTheLargestCopy(t *testing.T) {
	fMock := newHolder(true)
	if fMock.GetFetcher() == nil || fMock.GetFetcher().GetDownloadModel(videoAddress).Tool != "mock" {
		t.Fatalf("mock fetcher = %+v", fMock.GetFetcher())
	}
	f := newHolder(false)
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
	settings := f.GetSettings()
	if !slices.Equal(settings.Captions, []string{"sa", "en"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("the settings are %+v", settings)
	}
	f.UpdateConfig(map[string]string{"ytDlp": "", "ffmpeg": "", "copyMaxSizeMb": "many"})
	if f.GetSettings().CopyMaxSize != 20<<20 {
		t.Errorf("a size that is not a number changed CopyMaxSize to %d", f.GetSettings().CopyMaxSize)
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
	f := newHolder(false)
	f.UpdateConfig(map[string]string{
		"ytDlp": "/custom/yt-dlp", "automaticCaptions": "false", "copyMaxSizeMb": "20",
	})

	settings := f.GetSettings()
	if settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("automatic captions %t, largest copy %d; want false and %d", settings.IsAutomatic, settings.CopyMaxSize, int64(20<<20))
	}
}

func TestImportSettingsAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	f := newHolder(false)
	before := f.GetSettings()
	f.UpdateConfig(map[string]string{
		"yt_dlp": "/custom/yt-dlp", "automatic_captions": "false", "copy_max_size_mb": "20",
	})

	after := f.GetSettings()
	if after.IsAutomatic != before.IsAutomatic || after.CopyMaxSize != before.CopyMaxSize {
		t.Errorf("automatic captions %t and largest copy %d, want them unchanged at %t and %d",
			after.IsAutomatic, after.CopyMaxSize, before.IsAutomatic, before.CopyMaxSize)
	}
}

func newHolder(isMock bool) *Holder {
	tools := download.Tools{Runner: process.Runner{}, Files: tempfiles.Files{}, Paths: tempfiles.Files{}, Web: webpage.New()}
	return New(func(programs fetch.Config) ports.Fetcher { return download.Open(isMock, tools, programs) })
}
