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
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("upload a blob: %w", err)
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
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", err)
	}
	defer closeBody(response, &err)
	if err := checkTransferStatus("download", response); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, syncprotocol.MaxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("download a blob: %w", err)
	}
	if len(body) > syncprotocol.MaxBlobSize {
		return nil, fmt.Errorf("download a blob: the body is larger than %d bytes", syncprotocol.MaxBlobSize)
	}
	return body, nil
}

// GetBlobSpan downloads at most length bytes of the object from offset with one ranged GET, with no bearer and no
// protocol version. It accepts 206, and 200 only from offset 0; a 416 answer is the end of the object and returns
// no bytes. It returns fewer bytes than asked for only at the end of the object.
func GetBlobSpan(ctx context.Context, httpClient *http.Client, url string, offset, length int64) (span []byte, err error) {
	if offset < 0 || length <= 0 || length > syncprotocol.MaxBlobSize {
		return nil, fmt.Errorf("download a range: %d bytes from %d is not a range", length, offset)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare the download of a range: %w", err)
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download a range: %w", err)
	}
	defer closeBody(response, &err)
	if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return []byte{}, nil
	}
	if response.StatusCode != http.StatusPartialContent && (response.StatusCode != http.StatusOK || offset != 0) {
		return nil, fmt.Errorf("download a range: the storage answered %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, length+1))
	if err != nil {
		return nil, fmt.Errorf("download a range: %w", err)
	}
	if int64(len(body)) > length {
		return nil, fmt.Errorf("download a range: the body is larger than the %d bytes asked for", length)
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
