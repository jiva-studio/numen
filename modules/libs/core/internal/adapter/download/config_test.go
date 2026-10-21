package download

import (
	"slices"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if !d.CanUseAutomaticCaptions() {
		t.Error("automatic captions must be enabled by default")
	}
	if d.HasCopiesInVault() {
		t.Error("copies in vault must be disabled by default")
	}
	if d.CopyMaxSizeMB != DefaultCopyMaxSizeMB {
		t.Errorf("CopyMaxSizeMB = %d, want %d", d.CopyMaxSizeMB, DefaultCopyMaxSizeMB)
	}
	if d.CopyBytes() != int64(DefaultCopyMaxSizeMB)<<20 {
		t.Errorf("CopyBytes() = %d, want %d", d.CopyBytes(), int64(DefaultCopyMaxSizeMB)<<20)
	}
}

func TestCanUseAutomaticCaptions(t *testing.T) {
	on := true
	off := false
	if !(Config{AutomaticCaptions: nil}).CanUseAutomaticCaptions() {
		t.Error("nil AutomaticCaptions must be true")
	}
	if !(Config{AutomaticCaptions: &on}).CanUseAutomaticCaptions() {
		t.Error("&on AutomaticCaptions must be true")
	}
	if (Config{AutomaticCaptions: &off}).CanUseAutomaticCaptions() {
		t.Error("&off AutomaticCaptions must be false")
	}
}

func TestHasCopiesInVault(t *testing.T) {
	on := true
	off := false
	if (Config{CopiesToVault: nil}).HasCopiesInVault() {
		t.Error("nil CopiesToVault must be false")
	}
	if !(Config{CopiesToVault: &on}).HasCopiesInVault() {
		t.Error("&on CopiesToVault must be true")
	}
	if (Config{CopiesToVault: &off}).HasCopiesInVault() {
		t.Error("&off CopiesToVault must be false")
	}
}

func TestCopyBytes(t *testing.T) {
	if got := (Config{CopyMaxSizeMB: 0}).CopyBytes(); got != 0 {
		t.Errorf("CopyBytes() for 0 = %d, want 0", got)
	}
	if got := (Config{CopyMaxSizeMB: -5}).CopyBytes(); got != 0 {
		t.Errorf("CopyBytes() for negative = %d, want 0", got)
	}
	if got := (Config{CopyMaxSizeMB: 10}).CopyBytes(); got != 10<<20 {
		t.Errorf("CopyBytes() for 10 = %d, want %d", got, 10<<20)
	}
}

func TestCaptions(t *testing.T) {
	cfg := Config{Captions: []string{"en", "sa"}}
	if !slices.Equal(cfg.Captions, []string{"en", "sa"}) {
		t.Errorf("Captions = %v, want [en sa]", cfg.Captions)
	}
}
