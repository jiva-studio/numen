package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

// PutBlob uploads the body to a presigned URL with one PUT. It sends no bearer and no protocol version: the
// URL carries the permission.
func PutBlob(ctx context.Context, httpClient *http.Client, url string, body []byte) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare the upload of a blob: %w", err)
	}
	response, err := withoutRedirects(httpClient).Do(request)
	if err != nil {
		return fmt.Errorf("upload a blob: %w", redactURL(err))
	}
	defer closeBody(response, &err)
	return checkTransferStatus("upload", response)
}

// GetBlob downloads the body from a presigned URL with one GET, with no bearer and no protocol version. It
// refuses a body larger than syncprotocol.MaxBlobSize.
func GetBlob(ctx context.Context, httpClient *http.Client, url string) (blob []byte, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare the download of a blob: %w", err)
	}
	response, err := withoutRedirects(httpClient).Do(request)
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", redactURL(err))
	}
	defer closeBody(response, &err)
	if err := checkTransferStatus("download", response); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, syncprotocol.MaxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", redactURL(err))
	}
	if len(body) > syncprotocol.MaxBlobSize {
		return nil, fmt.Errorf("download a blob: the body is larger than %d bytes", syncprotocol.MaxBlobSize)
	}
	return body, nil
}

func checkTransferStatus(action string, response *http.Response) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s a blob: the storage answered %s", action, response.Status)
	}
	return nil
}

func closeBody(response *http.Response, err *error) {
	if closeErr := response.Body.Close(); closeErr != nil {
		*err = errors.Join(*err, closeErr)
	}
}

// PutPart uploads one part of a multipart upload to a presigned URL with one PUT. It sends no bearer and no
// protocol version.
func PutPart(ctx context.Context, httpClient *http.Client, url string, body []byte) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare the upload of a part: %w", err)
	}
	response, err := withoutRedirects(httpClient).Do(request)
	if err != nil {
		return fmt.Errorf("upload a part: %w", redactURL(err))
	}
	defer closeBody(response, &err)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("upload a part: the storage answered %s", response.Status)
	}
	return nil
}

// GetBlobRange starts a presigned GET at the byte from and returns the streamed body; the caller closes it. It
// accepts 206, or 200 when from is 0.
func GetBlobRange(ctx context.Context, httpClient *http.Client, url string, from int64) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare the download of a range: %w", err)
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-", from))
	response, err := withoutRedirects(httpClient).Do(request)
	if err != nil {
		return nil, fmt.Errorf("download a range: %w", redactURL(err))
	}
	return takeRangeBody(response, from)
}

func takeRangeBody(response *http.Response, from int64) (io.ReadCloser, error) {
	if response.StatusCode == http.StatusPartialContent || (response.StatusCode == http.StatusOK && from == 0) {
		return response.Body, nil
	}
	status := response.Status
	if closeErr := response.Body.Close(); closeErr != nil {
		return nil, fmt.Errorf("download a range: the storage answered %s: %w", status, closeErr)
	}
	return nil, fmt.Errorf("download a range: the storage answered %s", status)
}
