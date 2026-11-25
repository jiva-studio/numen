package fakeserver_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func TestRestore_ReadsTheSnapshot(t *testing.T) {
	first := fakeserver.New(fakeserver.Config{})
	kept, err := first.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	second := fakeserver.New(fakeserver.Config{})

	if err := second.Restore(kept); err != nil {
		t.Fatal(err)
	}

	again, err := second.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(kept, again) {
		t.Errorf("snapshot after restore = %s, want %s", again, kept)
	}
}

func TestRestore_RefusesWhatIsNotASnapshot(t *testing.T) {
	if err := fakeserver.New(fakeserver.Config{}).Restore([]byte("not json")); err == nil {
		t.Error("Restore of text that is not a snapshot succeeded, want an error")
	}
}

func newServerOver(tb testing.TB, bodyDir string) (*fakeserver.Server, string) {
	tb.Helper()
	server := fakeserver.New(fakeserver.Config{BodyDir: bodyDir})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	return server, served.URL
}

func TestSnapshot_KeepsBlobBodiesInTheBodyFolder(t *testing.T) {
	t.Parallel()
	bodyDir := t.TempDir()
	server, address := newServerOver(t, bodyDir)
	account := newOwnedAccount(t)
	e := newEnvOn(t, address, &account, account.addDevice(t, "First"))
	vault := e.newVault()
	body := []byte("a body that must stay out of the saved state")

	upload := e.upload(vault, body)

	kept, err := server.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(kept, body) || bytes.Contains(kept, []byte(base64.StdEncoding.EncodeToString(body))) {
		t.Errorf("the saved state holds the body of a blob: %s", kept)
	}
	if !bytes.HasPrefix(kept, []byte(fakeserver.SnapshotPrefix)) {
		t.Errorf("the saved state starts with %.20s, want %s", kept, fakeserver.SnapshotPrefix)
	}
	onDisk, err := os.ReadFile(filepath.Join(bodyDir, hex.EncodeToString(upload.GetBlobId()), "object"))
	if err != nil || !bytes.Equal(onDisk, body) {
		t.Errorf("the body file = %q, %v, want the body written once at upload", onDisk, err)
	}
}

func TestRestore_ServesAfterARestartABodyThatWasUploadedBefore(t *testing.T) {
	t.Parallel()
	bodyDir := t.TempDir()
	first, address := newServerOver(t, bodyDir)
	account := newOwnedAccount(t)
	e := newEnvOn(t, address, &account, account.addDevice(t, "First"))
	body := []byte("a body from before the restart")
	e.upload(e.newVault(), body)
	kept, err := first.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	second, secondAddress := newServerOver(t, bodyDir)
	if err := second.Restore(kept); err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, secondAddress+"/admin/blobs", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := io.ReadAll(response.Body)
	if err = errors.Join(err, response.Body.Close()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(listed, body) {
		t.Errorf("the restarted server lists %q, want it to hold the body", listed)
	}
}

func TestRestore_RefusesAStateOfTheOldLayout(t *testing.T) {
	t.Parallel()
	old := []byte(`{"chains":{},"bodies":{}}`)

	err := fakeserver.New(fakeserver.Config{}).Restore(old)

	if !errors.Is(err, fakeserver.ErrOldFormat) {
		t.Errorf("Restore of the old layout = %v, want ErrOldFormat", err)
	}
}
