package cloud_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func startMultipart(t *testing.T, _ *synctest.Computer, uploader *cloud.Uploader, vault []byte) transfer.Upload {
	t.Helper()
	up, err := uploader.Start(t.Context(), vault, syncprotocol.MaxBlobSize+1)
	if err != nil {
		t.Fatal(err)
	}
	return up
}

func newMultipartWorld(t *testing.T) (*synctest.Computer, *synctest.Server, *cloud.Uploader, []byte) {
	t.Helper()
	c, server := newWorld(t, fakeserver.Config{})
	vault := newVaultID(t)
	if _, err := c.Cloud.AppendCatalogEntry(t.Context(), c.NewCatalogEntry(t, nil, vault, true, []byte("sealed body"))); err != nil {
		t.Fatal(err)
	}
	return c, server, cloud.NewUploader(c.Cloud, c.Clock.Now), vault
}

func TestUploader_AnotherUploadGetsItsPartURLWhileTheRequestOfOneIsOpen(t *testing.T) {
	c, server, uploader, vault := newMultipartWorld(t)
	first, second := startMultipart(t, c, uploader, vault), startMultipart(t, c, uploader, vault)
	entered, release := make(chan struct{}), make(chan struct{})
	server.RunAfterNext("GetPartUrls", func() {
		close(entered)
		<-release
	})
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		if err := uploader.PutPart(t.Context(), first, 1, make([]byte, first.PartSize)); err != nil {
			t.Error(err)
		}
	}()
	<-entered
	defer func() {
		close(release)
		<-firstDone
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := uploader.PutPart(t.Context(), second, 1, make([]byte, second.PartSize)); err != nil {
			t.Error(err)
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("a part URL of another upload waited for the request of the first")
	}
}

func TestUploader_AsksForPartURLsAgainAfterTheUploadIsAborted(t *testing.T) {
	c, server, uploader, vault := newMultipartWorld(t)
	up := startMultipart(t, c, uploader, vault)
	if err := uploader.PutPart(t.Context(), up, 1, make([]byte, up.PartSize)); err != nil {
		t.Fatal(err)
	}
	asked := server.Fake.CountCalls(syncv1connect.BlobServiceGetPartUrlsProcedure)

	if err := uploader.Abort(t.Context(), up); err != nil {
		t.Fatal(err)
	}
	if err := uploader.PutPart(t.Context(), up, 1, make([]byte, up.PartSize)); err == nil {
		t.Fatal("a part of an aborted upload was stored")
	}

	if got := server.Fake.CountCalls(syncv1connect.BlobServiceGetPartUrlsProcedure) - asked; got != 1 {
		t.Errorf("part URL requests after the abort = %d, want 1: the cached URLs of the blob must be dropped", got)
	}
}
