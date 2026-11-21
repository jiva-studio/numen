package process

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPathPrefersTheProgramTheHostProvisioned(t *testing.T) {
	provisioned := filepath.Join(t.TempDir(), "yt-dlp")
	if err := os.WriteFile(provisioned, []byte("program"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_YT_DLP", provisioned)

	got, err := (Runner{}).LookPath("yt-dlp")

	if err != nil || got != provisioned {
		t.Errorf("LookPath = %q, %v, want %q", got, err, provisioned)
	}
}

func TestLookPathIgnoresAProvisionedProgramThatIsNotThere(t *testing.T) {
	t.Setenv("NUMEN_FFMPEG", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("PATH", t.TempDir())

	if got, err := (Runner{}).LookPath("ffmpeg"); err == nil {
		t.Errorf("LookPath = %q, want an error", got)
	}
}
