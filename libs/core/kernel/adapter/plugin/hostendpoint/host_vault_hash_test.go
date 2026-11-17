package hostendpoint_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const conflict = v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT

func TestAWriteOverTheHashOfWhatWasReadSucceedsAndAnswersTheNewHashAndAHigherVersion(t *testing.T) {
	_, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})
	read, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := read.Msg.GetStat().GetHash(), domain.HashContent([]byte("body")); got != want {
		t.Fatalf("the read answered the hash %q, want %q", got, want)
	}

	written, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("new body"), Baseline: read.Msg.GetStat(),
	}))
	if err != nil || written.Msg.GetError() != 0 {
		t.Fatalf("the write was answered %v (%v)", written.Msg, err)
	}

	if got, want := written.Msg.GetStat().GetHash(), domain.HashContent([]byte("new body")); got != want {
		t.Errorf("the write answered the hash %q, want %q", got, want)
	}
	if written.Msg.GetStat().GetVersion() <= read.Msg.GetStat().GetVersion() {
		t.Errorf("the write answered version %d after the read's %d", written.Msg.GetStat().GetVersion(), read.Msg.GetStat().GetVersion())
	}
	again, err := client.StatFile(t.Context(), connect.NewRequest(&v1.StatFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if again.Msg.GetStat().GetHash() != written.Msg.GetStat().GetHash() || again.Msg.GetStat().GetVersion() != written.Msg.GetStat().GetVersion() {
		t.Errorf("a stat after the write answered %v, the write answered %v", again.Msg.GetStat(), written.Msg.GetStat())
	}
}

func TestAWriteOverAStaleHashIsRefusedAsAConflict(t *testing.T) {
	_, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})
	read, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("first"), Baseline: read.Msg.GetStat(),
	})); err != nil {
		t.Fatal(err)
	}

	second, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("second"), Baseline: read.Msg.GetStat(),
	}))

	if err != nil || second.Msg.GetError() != conflict {
		t.Errorf("a write over the hash of an older read was answered %v (%v)", second.Msg, err)
	}
	got, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil || string(got.Msg.GetContent()) != "first" {
		t.Errorf("the file holds %q (%v)", got.Msg.GetContent(), err)
	}
}

func TestAnExternalChangeThatKeepsTheSizeAndTheTimeIsRefused(t *testing.T) {
	disk, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})
	target := filepath.Join(disk.current.Path, "samples", "A.md")
	read, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(0, read.Msg.GetStat().GetModTime())
	if err := os.WriteFile(target, []byte("BODY"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	withoutHash := &v1.HostFileStat{Path: "samples/A.md", Size: 4, ModTime: read.Msg.GetStat().GetModTime(), IsNote: true}

	byHash, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("mine"), Baseline: read.Msg.GetStat(),
	}))

	if err != nil || byHash.Msg.GetError() != conflict {
		t.Errorf("a write over a file that changed in content only was answered %v (%v)", byHash.Msg, err)
	}
	bySize, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("mine"), Baseline: withoutHash,
	}))
	if err != nil || bySize.Msg.GetError() != conflict {
		t.Errorf("a baseline without a hash is never the content of a file: answered %v (%v)", bySize.Msg, err)
	}
}
