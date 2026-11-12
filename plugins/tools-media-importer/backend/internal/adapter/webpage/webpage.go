// Package webpage fetches web pages over the network.
package webpage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const maxPageBytes = 8 << 20

const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// Client fetches the pages that web servers publish.
type Client struct {
	client *http.Client
}

// New returns a client that uses the proxy of the environment.
func New() *Client {
	dialer := new(net.Dialer)
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="net.Dialer fields are captured in closure and not observable without waiting 30s"
	dialer.Timeout = 30 * time.Second
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="net.Dialer fields are captured in closure and not observable without waiting 30s"
	dialer.KeepAlive = 30 * time.Second

	transport := new(http.Transport)
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = dialer.DialContext
	transport.ForceAttemptHTTP2 = true
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ExpectContinueTimeout = 1 * time.Second

	client := new(http.Client)
	client.Transport = transport
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("this address sends the fetch on and on")
		}
		return nil
	}
	return &Client{client: client}
}

type limitedBody struct {
	io.Reader
	io.Closer
}

// Fetch requests the page at target. The reader it returns ends after 8 MiB, and closing it closes the answer.
func (c *Client) Fetch(ctx context.Context, target string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	answer, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if answer.StatusCode != http.StatusOK {
		return nil, errors.Join(fmt.Errorf("%s answered %s", target, answer.Status), answer.Body.Close())
	}
	return limitedBody{Reader: io.LimitReader(answer.Body, maxPageBytes), Closer: answer.Body}, nil
}
