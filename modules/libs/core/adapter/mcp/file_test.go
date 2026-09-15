package mcp_test

import (
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// run is what file_read answers with.
type run struct {
	Text    string `json:"text"`
	Start   int    `json:"start"`
	Length  int    `json:"length"`
	Size    int    `json:"size"`
	Refused string `json:"refused"`
}

func runOf(t *testing.T, s *sdk.ClientSession, path string, start, length int) run {
	t.Helper()
	return call[run](t, s, "file_read", map[string]any{
		"path": path, "start": start, "length": length,
	})
}

// A transcribed lecture is a file somebody put in the vault, and the tools that
// read a note do not reach it.
func TestFileReadReachesAFileTheVaultDoesNotHoldAsANote(t *testing.T) {
	session, _ := newSession(t, map[string]string{
		"lectures/kinetics.txt": "The first law, as spoken.\n",
	})

	got := runOf(t, session, "lectures/kinetics.txt", 0, 0)
	if got.Text != "The first law, as spoken.\n" {
		t.Errorf("the file came back as %q, refused %q", got.Text, got.Refused)
	}
	if got.Size != len(got.Text) {
		t.Errorf("the file was said to be %d bytes", got.Size)
	}
}

// A transcript is longer than one answer carries, and what the answer says about
// the run it gave is what the next call is asked with.
func TestFileReadReadsALongFileARunAtATime(t *testing.T) {
	whole := strings.Repeat("one line of what was said\n", 400)
	session, _ := newSession(t, map[string]string{"lecture.txt": whole})

	first := runOf(t, session, "lecture.txt", 0, 60)
	if first.Text != whole[:60] {
		t.Fatalf("the first run came back as %q", first.Text)
	}
	next := runOf(t, session, "lecture.txt", first.Start+first.Length, 60)
	if next.Text != whole[60:120] {
		t.Errorf("reading on gave %q", next.Text)
	}
}

// A path with nothing behind it is an answer and not an error: a file may have
// been removed since an agent last saw it.
func TestFileReadSaysWhenThereIsNoFileAtThePath(t *testing.T) {
	session, _ := newSession(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := runOf(t, session, "gone.txt", 0, 0)
	if got.Refused == "" {
		t.Errorf("a path with no file behind it answered with %q", got.Text)
	}
}

// What the vault passes over is passed over here, so an agent on this endpoint
// reaches no further into the folder than the application itself does.
func TestFileReadPassesOverWhatTheVaultDoes(t *testing.T) {
	session, _ := newSession(t, map[string]string{
		".env":              "SECRET=1\n",
		"notes/lecture.txt": "said",
	})

	got := runOf(t, session, ".env", 0, 0)
	if got.Text != "" || got.Refused == "" {
		t.Errorf("a name the vault passes over answered with %+v", got)
	}
}

// The path is from the vault root, and a call naming another place is a mistake
// worth hearing about.
func TestFileReadRefusesAPathThatLeavesTheVault(t *testing.T) {
	session, v := newSession(t, map[string]string{"Entropy.md": "# Entropy\n"})

	for _, path := range []string{"../secret.txt", "notes/../../secret.txt", v.Path + "/Entropy.md"} {
		if said := getRefusal(t, session, "file_read", map[string]any{"path": path}); said == "" {
			t.Errorf("read %q: nothing said about a path outside the vault", path)
		}
	}
}
