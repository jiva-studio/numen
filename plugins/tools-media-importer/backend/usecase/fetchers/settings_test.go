package fetchers

import (
	"slices"
	"testing"
)

func TestTheSettingsNameThePrograms(t *testing.T) {
	values := parseValues(map[string]string{"ytDlp": "/custom/yt-dlp", "ffmpeg": "/custom/ffmpeg"}, getDefaultValues())
	if !slices.Equal(values.Programs.Video.Command, []string{"/custom/yt-dlp"}) || !slices.Equal(values.Programs.Sound.Command, []string{"/custom/ffmpeg"}) {
		t.Errorf("the programs are %v and %v", values.Programs.Video.Command, values.Programs.Sound.Command)
	}
	values = parseValues(map[string]string{"ytDlp": "", "ffmpeg": ""}, values)
	if len(values.Programs.Video.Command) != 0 || len(values.Programs.Sound.Command) != 0 {
		t.Errorf("cleared settings leave %v and %v", values.Programs.Video.Command, values.Programs.Sound.Command)
	}
}

func TestTheProgramsAreNotReadFromTheSnakeCaseKey(t *testing.T) {
	before := getDefaultValues()
	after := parseValues(map[string]string{"yt_dlp": "/custom/yt-dlp"}, before)
	if !slices.Equal(after.Programs.Video.Command, before.Programs.Video.Command) {
		t.Errorf("the video program is %v, want it unchanged at %v", after.Programs.Video.Command, before.Programs.Video.Command)
	}
}
