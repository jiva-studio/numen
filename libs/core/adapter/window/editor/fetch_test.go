package editor

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

func fetch(t testing.TB, client *http.Client, address string) fetched {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	return send(t, client, request)
}

func send(t testing.TB, client *http.Client, request *http.Request) fetched {
	t.Helper()
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
