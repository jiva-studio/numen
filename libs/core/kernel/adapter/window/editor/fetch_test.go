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

func fetch(tb testing.TB, client *http.Client, address string) fetched {
	tb.Helper()
	request, err := http.NewRequestWithContext(tb.Context(), http.MethodGet, address, nil)
	if err != nil {
		tb.Fatal(err)
	}
	return send(tb, client, request)
}

func send(tb testing.TB, client *http.Client, request *http.Request) fetched {
	tb.Helper()
	response, err := client.Do(request)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			tb.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		tb.Fatal(err)
	}
	return fetched{StatusCode: response.StatusCode, Header: response.Header, Body: body}
}
