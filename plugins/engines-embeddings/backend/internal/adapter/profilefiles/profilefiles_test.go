package profilefiles

import "testing"

func TestTheDefaultDirectoryIsTheProfilesFolderOfTheDataDirectory(t *testing.T) {
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "/tmp/numen-data")
	if got := GetDefaultDirectory().path; got != "/tmp/numen-data/profiles" {
		t.Fatalf("directory = %q", got)
	}
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "")
	if got := GetDefaultDirectory().path; got != "" {
		t.Fatalf("directory without a data directory = %q, want empty", got)
	}
}
