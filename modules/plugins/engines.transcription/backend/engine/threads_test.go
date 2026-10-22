package engine

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func TestOnlyTheEncoderConsumesMoreThanOneThread(t *testing.T) {
	got := Defaults().sessionThreads()
	want := sessionThreads{encoder: onnx.GetDefaultThreads(), decoder: 1, joiner: 1, segmenter: 1}
	if got != want {
		t.Errorf("the sessions take %+v threads, want %+v", got, want)
	}
}

func TestTheEncoderConsumesTheThreadsItIsGiven(t *testing.T) {
	cfg := Defaults()
	cfg.Threads = 3
	got := cfg.sessionThreads()
	want := sessionThreads{encoder: 3, decoder: 1, joiner: 1, segmenter: 1}
	if got != want {
		t.Errorf("the sessions take %+v threads, want %+v", got, want)
	}
}
