package syncprotocol

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// PutBlob uploads the body to a presigned URL with one PUT. It sends no bearer and no protocol version: the
// URL carries the permission.
func PutBlob(ctx context.Context, httpClient *http.Client, url string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare the upload of a blob: %w", err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("upload a blob: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	return checkTransferStatus("upload", response)
}

// GetBlob downloads the body from a presigned URL with one GET, with no bearer and no protocol version. It
// refuses a body larger than MaxBlobSize.
func GetBlob(ctx context.Context, httpClient *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare the download of a blob: %w", err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if err := checkTransferStatus("download", response); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", err)
	}
	if len(body) > MaxBlobSize {
		return nil, fmt.Errorf("download a blob: the body is larger than %d bytes", MaxBlobSize)
	}
	return body, nil
}

func checkTransferStatus(action string, response *http.Response) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s a blob: the storage answered %s", action, response.Status)
	}
	return nil
}
