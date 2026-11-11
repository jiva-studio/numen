package engine

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

func TestOnlyTheEncoderConsumesMoreThanOneThread(t *testing.T) {
	got := getSessionThreads(settings.Defaults())
	want := sessionThreads{encoder: onnx.GetDefaultThreads(), decoder: 1, joiner: 1, segmenter: 1}
	if got != want {
		t.Errorf("the sessions take %+v threads, want %+v", got, want)
	}
}

func TestTheEncoderConsumesTheThreadsItIsGiven(t *testing.T) {
	for _, count := range []int{1, 3} {
		cfg := settings.Defaults()
		cfg.Threads = count
		got := getSessionThreads(cfg)
		want := sessionThreads{encoder: count, decoder: 1, joiner: 1, segmenter: 1}
		if got != want {
			t.Errorf("the sessions take %+v threads, want %+v", got, want)
		}
	}
}
