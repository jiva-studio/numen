package download

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

func TestTheSoundAloneIsDownloadedWithoutAVideoToMerge(t *testing.T) {
	if got := getFormatArguments(fetch.MediaAudio); !slices.Equal(got, []string{"-f", soundFormat}) {
		t.Errorf("the sound is downloaded with %v", got)
	}
	if got := getFormatArguments(fetch.MediaVideo); !slices.Contains(got, copyFormat) || !slices.Contains(got, "--merge-output-format") {
		t.Errorf("the video is downloaded with %v", got)
	}
}
