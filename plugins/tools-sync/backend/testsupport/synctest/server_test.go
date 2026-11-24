package synctest_test

import (
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func getStatus(t *testing.T, url string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode
}

func TestALostResponseWaitsOutTheTimeTheServerIsDown(t *testing.T) {
	server := synctest.NewServer(t, synctest.NewClock(synctest.GetStart()), fakeserver.Config{})
	server.LoseNextResponse("/ping")
	server.SetDown(true)
	getStatus(t, server.URL+"/ping")
	server.SetDown(false)

	if got := getStatus(t, server.URL+"/ping"); got != http.StatusServiceUnavailable {
		t.Errorf("the first request after the outage answered %d, want 503 of the lost response", got)
	}
	if got := getStatus(t, server.URL+"/ping"); got == http.StatusServiceUnavailable {
		t.Errorf("the second request answered %d, want the response of the server", got)
	}
}
