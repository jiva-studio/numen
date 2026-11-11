// Package feed fetches the release manifest from the endpoint that publishes it.
package feed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/domain/release"
)

// requestTimeout bounds one fetch of the manifest, connection and body included.
const requestTimeout = 10 * time.Second

// maxManifestBytes bounds the body read from the endpoint.
const maxManifestBytes = 1 << 20

// ErrNotWebAddress is an endpoint that is not an absolute http or https address.
var ErrNotWebAddress = errors.New("the endpoint is not a web address")

// ErrStatus is an answer of the endpoint that is not 200.
var ErrStatus = errors.New("the endpoint did not answer 200")

// Client fetches release manifests.
type Client struct {
	http *http.Client
}

// New returns a client that fetches over the network with the proxy of the environment.
func New() *Client {
	return &Client{http: &http.Client{Timeout: requestTimeout}}
}

// CheckEndpoint reports whether the endpoint is an absolute http or https address.
func CheckEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%w: %q", ErrNotWebAddress, endpoint)
	}
	return nil
}

// Fetch requests the manifest from the endpoint. It fails for an endpoint that is no web address, for
// an answer other than 200 and for a body that is no manifest.
func (c *Client) Fetch(ctx context.Context, endpoint string) (manifest release.Manifest, err error) {
	if err := CheckEndpoint(endpoint); err != nil {
		return release.Manifest{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	// gomutants:disable-next-line BRANCH_IF reason="CheckEndpoint already validates url.Parse for the endpoint."
	if err != nil {
		return release.Manifest{}, fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return release.Manifest{}, fmt.Errorf("request release manifest: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return release.Manifest{}, fmt.Errorf("%w: %d", ErrStatus, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes))
	if err != nil {
		return release.Manifest{}, fmt.Errorf("read release manifest: %w", err)
	}
	return release.Read(body)
}
