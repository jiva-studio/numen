package hosttest_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestAWriteOverTheHashOfWhatWasReadReturnsTheNewHashAndAHigherVersion(t *testing.T) {
	ctx := t.Context()
	v := hosttest.New("Physics")
	v.WriteFile("a.md", "one")
	_, read, err := v.Read(ctx, "a.md")
	if err != nil {
		t.Fatal(err)
	}
	if read.Hash != domain.HashContent([]byte("one")) {
		t.Fatalf("the read returned the hash %q", read.Hash)
	}

	written, err := v.Write(ctx, "a.md", []byte("two"), read)

	if err != nil {
		t.Fatal(err)
	}
	if written.Hash != domain.HashContent([]byte("two")) || written.Version <= read.Version {
		t.Errorf("the write returned %+v after the read %+v", written, read)
	}
	statted, err := v.StatFile(ctx, "a.md")
	if err != nil || statted.Hash != written.Hash || statted.Version != written.Version {
		t.Errorf("a stat after the write returned %+v (%v), the write %+v", statted, err, written)
	}
}

func TestAWriteOverAStaleHashIsStale(t *testing.T) {
	ctx := t.Context()
	v := hosttest.New("Physics")
	v.WriteFile("a.md", "one")
	_, read, err := v.Read(ctx, "a.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write(ctx, "a.md", []byte("two"), read); err != nil {
		t.Fatal(err)
	}

	_, err = v.Write(ctx, "a.md", []byte("three"), read)

	if !errors.Is(err, host.ErrStale) {
		t.Errorf("got %v", err)
	}
}

func TestAChangeThatKeepsSizeAndTimeOnDiskIsStaleAndSoIsABaselineWithoutAHash(t *testing.T) {
	ctx := t.Context()
	v := hosttest.CreateOnDisk(t, "Physics", map[string]string{"a.md": "body"})
	_, read, err := v.Read(ctx, "a.md")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(v.Vault().Path, "a.md")
	if err := os.WriteFile(target, []byte("BODY"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, read.ModTime, read.ModTime); err != nil {
		t.Fatal(err)
	}
	withoutHash := read
	withoutHash.Hash = ""

	_, byHash := v.Write(ctx, "a.md", []byte("mine"), read)
	_, bySize := v.Write(ctx, "a.md", []byte("mine"), withoutHash)

	if !errors.Is(byHash, host.ErrStale) || !errors.Is(bySize, host.ErrStale) {
		t.Errorf("the write over the hash returned %v and the one without a hash %v", byHash, bySize)
	}
}
