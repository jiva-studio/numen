package fakeserver_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	bigSize  = syncprotocol.MaxBlobSize + 1
	partSize = syncprotocol.MinPartSize
)

func newBigBody() []byte {
	body := make([]byte, bigSize)
	for i := range body {
		body[i] = byte(i * 7)
	}
	return body
}

func (e *env) putParts(vault, blob []byte, body []byte, numbers ...uint32) {
	e.tb.Helper()
	urls, err := e.blob.GetPartUrls(e.tb.Context(), vault, blob, numbers)
	if err != nil {
		e.tb.Fatal(err)
	}
	for _, one := range urls {
		from := int(one.GetPartNumber()-1) * partSize
		to := min(from+partSize, len(body))
		if err := syncclient.PutPart(e.tb.Context(), http.DefaultClient, one.GetUrl(), body[from:to]); err != nil {
			e.tb.Fatal(err)
		}
	}
}

func TestMultipart_JoinsPartsIntoOneObjectThatAnotherDeviceReadsByRange(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := newBigBody()
	reserved, err := e.blob.ReserveMultipartBlob(t.Context(), vault, uint64(len(body)), partSize)
	if err != nil {
		t.Fatal(err)
	}
	count := reserved.GetPartCount()
	numbers := make([]uint32, 0, count)
	for number := uint32(1); number <= count; number++ {
		numbers = append(numbers, number)
	}

	for batch := 0; batch < len(numbers); batch += syncprotocol.MaxPartURLs {
		e.putParts(vault, reserved.GetBlobId(), body, numbers[batch:min(batch+syncprotocol.MaxPartURLs, len(numbers))]...)
	}
	if err := e.blob.CompleteMultipartBlob(t.Context(), vault, reserved.GetBlobId()); err != nil {
		t.Fatal(err)
	}
	e.commit(vault, e.newRevisionOf(vault, newID(16), nil, "big.bin", &syncv1.BlobUpload{BlobId: reserved.GetBlobId()}, body))

	downloads, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{reserved.GetBlobId()})
	if err != nil || !downloads[0].GetFound() {
		t.Fatalf("downloads = %v, error = %v, want the blob found", downloads, err)
	}
	from := int64(len(body) / 2)
	stream, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, downloads[0].GetUrl(), from)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(stream)
	if err != nil || !bytes.Equal(got, body[from:]) {
		t.Errorf("range read %d bytes, error = %v, want the %d bytes from the middle", len(got), err, len(body)-int(from))
	}
	if err := stream.Close(); err != nil {
		t.Error(err)
	}
}

func TestMultipart_ListsTheHeldPartsAndNamesTheMissingOnesAtComplete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := newBigBody()
	reserved, err := e.blob.ReserveMultipartBlob(t.Context(), vault, uint64(len(body)), partSize)
	if err != nil {
		t.Fatal(err)
	}

	e.putParts(vault, reserved.GetBlobId(), body, 3, 1)

	parts, err := e.blob.ListParts(t.Context(), vault, reserved.GetBlobId())
	if err != nil || len(parts) != 2 || parts[0].GetPartNumber() != 1 || parts[1].GetPartNumber() != 3 {
		t.Fatalf("parts = %v, error = %v, want parts 1 and 3 in order", parts, err)
	}
	err = e.blob.CompleteMultipartBlob(t.Context(), vault, reserved.GetBlobId())
	var incomplete syncprotocol.MultipartIncompleteError
	if !errors.As(err, &incomplete) || len(incomplete.Missing) != int(reserved.GetPartCount())-2 || incomplete.Missing[0] != 2 {
		t.Errorf("error = %v, want the parts but 1 and 3 missing", err)
	}
}

func TestMultipart_AbortFreesTheUploadAndRepeatsAsASuccess(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	reserved, err := e.blob.ReserveMultipartBlob(t.Context(), vault, bigSize, partSize)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := e.blob.AbortMultipartBlob(t.Context(), vault, reserved.GetBlobId()); err != nil {
			t.Fatalf("abort: %v", err)
		}
	}

	if _, err := e.blob.ListParts(t.Context(), vault, reserved.GetBlobId()); err == nil {
		t.Error("ListParts of an aborted upload gave no error")
	}
}

func TestMultipart_RefusesSizesOutsideTheLimits(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()

	if _, err := e.blob.ReserveMultipartBlob(t.Context(), vault, syncprotocol.MaxBlobSize, partSize); err == nil {
		t.Error("a size of one single upload was reserved as multipart")
	}
	if _, err := e.blob.ReserveMultipartBlob(t.Context(), vault, bigSize, partSize-1); err == nil {
		t.Error("a part under the minimum was accepted")
	}
}
