package webpage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type paddedHTMLReader struct {
	prefix string
	pos    int
	total  int
}

func (r *paddedHTMLReader) Read(p []byte) (int, error) {
	if r.pos >= r.total {
		return 0, io.EOF
	}
	limit := min(len(p), r.total-r.pos)
	for i := range limit {
		idx := r.pos + i
		if idx < len(r.prefix) {
			p[i] = r.prefix[idx]
		} else {
			p[i] = ' '
		}
	}
	r.pos += limit
	return limit, nil
}

func (r *paddedHTMLReader) Close() error { return nil }

func TestClientConfigurationAndBodyLimit(t *testing.T) {
	pages := New()
	if pages.client.Timeout != 30*time.Second {
		t.Errorf("client.Timeout = %v, want 30s", pages.client.Timeout)
	}
	tr, ok := pages.client.Transport.(*http.Transport)
	if !ok || tr == nil {
		t.Fatalf("client.Transport = %T, want *http.Transport", pages.client.Transport)
	}
	if tr.Proxy == nil || tr.DialContext == nil || !tr.ForceAttemptHTTP2 {
		t.Errorf("transport missing Proxy, DialContext, or ForceAttemptHTTP2: %+v", tr)
	}
	if tr.MaxIdleConns != 100 || tr.MaxIdleConnsPerHost != 20 {
		t.Errorf("idle conns = (%d, %d), want (100, 20)", tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 90*time.Second || tr.TLSHandshakeTimeout != 10*time.Second || tr.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("timeouts = (%v, %v, %v), want (90s, 10s, 1s)", tr.IdleConnTimeout, tr.TLSHandshakeTimeout, tr.ExpectContinueTimeout)
	}
	body := &paddedHTMLReader{prefix: "<html></html>", total: 9 << 20}
	pages.client = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body}, nil
		}),
	}
	page, err := pages.Fetch(t.Context(), "https://example.com/post")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(page); err != nil {
		t.Fatal(err)
	}
	if err := page.Close(); err != nil {
		t.Fatal(err)
	}
	if body.pos != 8<<20 {
		t.Errorf("Fetch read %d bytes, want %d", body.pos, 8<<20)
	}
}

func TestFetchRefusesAnAnswerOtherThanOK(t *testing.T) {
	web := httptest.NewServer(http.NotFoundHandler())
	defer web.Close()
	if _, err := New().Fetch(t.Context(), web.URL); err == nil {
		t.Error("Fetch of a missing page = nil error, want one")
	}
}

func TestFollowARedirectToAnAddressOnThisMachine(t *testing.T) {
	pages := New()
	via := make([]*http.Request, 9)
	for _, raw := range []string{"http://127.0.0.1/secret", "http://192.168.1.10/"} {
		target, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := pages.client.CheckRedirect(&http.Request{URL: target}, via); err != nil {
			t.Errorf("CheckRedirect(%s) after 9 hops = %v, want nil", raw, err)
		}
		if err := pages.client.CheckRedirect(&http.Request{URL: target}, nil); err != nil {
			t.Errorf("CheckRedirect(%s) = %v, want nil", raw, err)
		}
	}
}

func TestStopAfterTenRedirects(t *testing.T) {
	target, err := url.Parse("https://example.com/next")
	if err != nil {
		t.Fatal(err)
	}
	if err := New().client.CheckRedirect(&http.Request{URL: target}, make([]*http.Request, 10)); err == nil {
		t.Error("CheckRedirect after 10 hops = nil, want an error")
	}
}
