package keychain

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/zalando/go-keyring"
)

func TestADeleteOfAKeyHeldNowhereIsNoKey(t *testing.T) {
	useNoKeychain(t)
	store := newTestStore(t.TempDir())
	if err := store.DeleteKey(t.Context(), entryName); err == nil {
		t.Error("a delete of a key held nowhere succeeded")
	}
}

func TestADeleteRemovesTheKeyFromTheKeychainAndFromTheFile(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(credentialsPath(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath(dir), []byte(`{"`+entryName+`":"`+valueOne+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(dir)
	if err := store.SetKey(t.Context(), entryName, valueTwo); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteKey(t.Context(), entryName); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetKey(t.Context(), entryName); err == nil {
		t.Error("a deleted key is still read")
	}
}

func TestACredentialsFileThatIsNotJSONFailsWithoutQuotingItsContent(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(credentialsPath(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath(dir), []byte(`{"`+entryName+`": `+valueOne), 0o600); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(dir)
	_, err := store.GetKey(t.Context(), entryName)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON at byte") || strings.Contains(err.Error(), valueOne) {
		t.Errorf("GetKey syntax error = %v", err)
	}
	if err := store.DeleteKey(t.Context(), entryName); err == nil || errors.Is(err, plugin.ErrNoKey) || !strings.Contains(err.Error(), "invalid JSON at byte") {
		t.Errorf("DeleteKey syntax error = %v", err)
	}

	if err := os.WriteFile(credentialsPath(dir), []byte(`{"`+entryName+`":"`+valueOne+`","bad":123}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetKey(t.Context(), entryName)
	if err == nil || !strings.Contains(err.Error(), "a value has the wrong type") || strings.Contains(err.Error(), valueOne) {
		t.Errorf("GetKey type error = %v", err)
	}

	dirAsFile := t.TempDir()
	if err := os.MkdirAll(credentialsPath(dirAsFile), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = newTestStore(dirAsFile).GetKey(t.Context(), entryName)
	if pathErr, ok := errors.AsType[*fs.PathError](err); !ok || pathErr == nil {
		t.Errorf("GetKey read error = %v, want wrapped *fs.PathError", err)
	}
}
