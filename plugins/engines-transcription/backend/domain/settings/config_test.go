package settings

import "testing"

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Runtime != "" || cfg.Dir != "" || cfg.Threads != 0 || cfg.Workers != 0 || cfg.Progress != nil || !cfg.ShouldDownload {
		t.Errorf("Defaults() = %+v", cfg)
	}
	if cfg.Model != GetModels()[0] {
		t.Errorf("Defaults().Model = %+v, want %+v", cfg.Model, GetModels()[0])
	}
	wantSegmenter := SegmenterModel{Name: "silero-vad", Repo: segmenterDefault}
	if cfg.Segmenter != wantSegmenter {
		t.Errorf("Defaults().Segmenter = %+v, want %+v", cfg.Segmenter, wantSegmenter)
	}
}

func TestUnsetThreadsTakeTheCoresGiven(t *testing.T) {
	if got := Defaults().GetThreads(7); got != 7 {
		t.Errorf("unset threads gave %d, want 7", got)
	}
	cfg := Defaults()
	cfg.Threads = 3
	if got := cfg.GetThreads(7); got != 3 {
		t.Errorf("3 threads gave %d", got)
	}
}
