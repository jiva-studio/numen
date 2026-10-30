package shell

import (
	"io"
	"net/http"
	"testing"
)

// fetched is a response whose body has been read and closed.
type fetched struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// fetch sends a GET request bound to the test context, reads the whole response
// body and closes it. It fails the test when any step fails.
func fetch(t testing.TB, client *http.Client, address string) fetched {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return fetched{StatusCode: response.StatusCode, Header: response.Header, Body: body}
}
