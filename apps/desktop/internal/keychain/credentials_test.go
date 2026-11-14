package keychain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	_, err := newTestStore(dir).GetKey(t.Context(), entryName)
	if err == nil || strings.Contains(err.Error(), valueOne) {
		t.Errorf("error = %v", err)
	}
}
