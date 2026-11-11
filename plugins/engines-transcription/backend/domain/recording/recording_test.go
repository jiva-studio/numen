package recording

import "testing"

func TestIsRecordingReadsTheExtensionInAnyCase(t *testing.T) {
	for file, want := range map[string]bool{
		"assets/lecture.wav": true,
		"assets/Lecture.MP3": true,
		"clips/tides.mkv":    true,
		"notes/plan.md":      false,
		"assets/lecture":     false,
		"wav":                false,
		"assets/cover.png":   false,
		"archive.wav.zip":    false,
		"folder.mp4/readme":  false,
	} {
		if got := IsRecording(file); got != want {
			t.Errorf("IsRecording(%q) = %v, want %v", file, got, want)
		}
	}
}
