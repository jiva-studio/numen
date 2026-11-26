package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestMakeIdentity_MakesTheSameWordsForTheSameLabel(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	one, err := makeIdentity("a scenario 1", now)
	if err != nil {
		t.Fatal(err)
	}
	again, err := makeIdentity("a scenario 1", now)
	if err != nil {
		t.Fatal(err)
	}
	other, err := makeIdentity("a scenario 2", now)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(one.words.List(), again.words.List()) || one.chain.Root().Fingerprint() != again.chain.Root().Fingerprint() {
		t.Error("the same label made another account")
	}
	if slices.Equal(one.words.List(), other.words.List()) {
		t.Error("two labels made the same words")
	}
}

func TestMakeIdentity_ListsTheComputersInTheChain(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	made, err := makeIdentity("a scenario", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := made.first.Len(); got != 2 {
		t.Errorf("the chain of the first sign-in has %d entries, want 2", got)
	}
	if got := len(made.chain.ListDevices()); got != len(getComputerNames()) {
		t.Errorf("the chain lists %d devices, want %d", got, len(getComputerNames()))
	}
	if err := made.chain.VerifyAt(now); err != nil {
		t.Errorf("the chain does not verify at the time of its making: %v", err)
	}
	if !made.chain.IsExtensionOf(made.first) {
		t.Error("the full chain does not extend the chain of the first sign-in")
	}
}

func TestBuildSeedFile_OpensTheKeysOfEveryComputerForItsDevice(t *testing.T) {
	made, err := makeIdentity("a scenario", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	file, err := buildSeedFile(made)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(file.Chain)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range getComputerNames() {
		raw, err := base64.RawURLEncoding.DecodeString(file.Computers[name].Credentials[keyName])
		if err != nil {
			t.Fatal(err)
		}
		record, err := identity.ParseRecord(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := identity.NewState(record, chain); err != nil {
			t.Errorf("%s: its keys do not open for the chain: %v", name, err)
		}
	}
}

func TestEncodeQuery_SortsKeysAndEscapesSpaces(t *testing.T) {
	got := encodeQuery(url.Values{"b": {"x y"}, "a": {"1"}})
	if want := "a=1&b=x%20y"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func writeText(t *testing.T, w http.ResponseWriter, text string) {
	t.Helper()
	if _, err := w.Write([]byte(text)); err != nil {
		t.Error(err)
	}
}

func TestFindText_FailsOnAnObjectWithTheText(t *testing.T) {
	served := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list-type") == "2" {
			writeText(t, w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>one</Key><LastModified>2026-11-26T12:00:00Z</LastModified></Contents></ListBucketResult>`)
			return
		}
		writeText(t, w, "sealed bytes with milk inside")
	}))
	defer served.Close()
	b := bucket{endpoint: served.URL, name: "numen-sync", region: "us-east-1", access: "a", secret: "s", client: served.Client()}
	if _, err := b.findText(t.Context(), []byte("milk"), time.Unix(0, 0)); err == nil {
		t.Error("a text in an object was not found")
	}
	if read, err := b.findText(t.Context(), []byte("bread"), time.Unix(0, 0)); err != nil || read != 1 {
		t.Errorf("read %d objects, error %v; want 1 and no error", read, err)
	}
	if _, err := b.findText(t.Context(), []byte("bread"), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Error("a bucket with no object since then passed")
	}
}
