package openai

import "net/http"

// GetHTTPClient is the client every request is sent with.
func (c *Client) GetHTTPClient() *http.Client { return c.http }
