package download

import (
	"slices"
	"testing"
)

func TestTheSoundAloneIsDownloadedWithoutAVideoToMerge(t *testing.T) {
	if got := getFormatArguments(MediaAudio); !slices.Equal(got, []string{"-f", soundFormat}) {
		t.Errorf("the sound is downloaded with %v", got)
	}
	if got := getFormatArguments(MediaVideo); !slices.Contains(got, copyFormat) || !slices.Contains(got, "--merge-output-format") {
		t.Errorf("the video is downloaded with %v", got)
	}
}
