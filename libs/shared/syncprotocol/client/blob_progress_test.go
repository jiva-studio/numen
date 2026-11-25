package client_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

const (
	idle  = 150 * time.Millisecond
	steps = 8
	pause = 50 * time.Millisecond
)

func serveSlowly(t *testing.T, stallAfter int) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if _, err := io.Copy(io.Discard, r.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		for i := range steps {
			if i == stallAfter {
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
				return
			}
			if _, err := w.Write(bytes.Repeat([]byte{1}, 1024)); err != nil {
				return
			}
			w.(http.Flusher).Flush()
			time.Sleep(pause)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestGetBlob_ABodyThatKeepsArrivingIsNotCutByAnyTotalTime(t *testing.T) {
	t.Parallel()
	address := serveSlowly(t, -1)
	client := syncclient.NewBlobHTTPClient(address, idle)

	body, err := syncclient.GetBlob(t.Context(), client, address+"/blob")

	if err != nil || len(body) != steps*1024 {
		t.Fatalf("got %d bytes and %v, want %d bytes: the body took %v, longer than the idle time %v", len(body), err, steps*1024, steps*pause, idle)
	}
}

func TestGetBlob_ABodyThatStopsArrivingEndsAfterTheIdleTime(t *testing.T) {
	t.Parallel()
	address := serveSlowly(t, 3)
	client := syncclient.NewBlobHTTPClient(address, idle)
	started := time.Now()

	_, err := syncclient.GetBlob(t.Context(), client, address+"/blob")

	if !errors.Is(err, syncclient.ErrTransferStalled) {
		t.Fatalf("error = %v, want ErrTransferStalled", err)
	}
	if took := time.Since(started); took > 3*time.Second {
		t.Errorf("the stalled read took %v", took)
	}
}

func TestGetBlobRange_ABodyThatStopsArrivingEndsTheReadOfItsBody(t *testing.T) {
	t.Parallel()
	address := serveSlowly(t, 2)
	client := syncclient.NewBlobHTTPClient(address, idle)
	body, err := syncclient.GetBlobRange(t.Context(), client, address+"/blob", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()

	_, err = io.ReadAll(body)

	if !errors.Is(err, syncclient.ErrTransferStalled) {
		t.Errorf("error = %v, want ErrTransferStalled", err)
	}
}

type trickle struct {
	left int
}

func (t *trickle) Read(p []byte) (int, error) {
	if t.left == 0 {
		return 0, io.EOF
	}
	time.Sleep(pause)
	t.left--
	return copy(p, bytes.Repeat([]byte{2}, 1024)), nil
}

func TestBlobClient_AnUploadThatTakesLongerThanTheIdleTimeButKeepsMovingIsSent(t *testing.T) {
	t.Parallel()
	address := serveSlowly(t, -1)
	client := syncclient.NewBlobHTTPClient(address, idle)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, address+"/blob", io.NopCloser(&trickle{left: steps}))
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("a slow upload that never stood still failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %s", response.Status)
	}
}
