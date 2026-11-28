package synctest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const (
	bucketAccessKey = "numendev"
	bucketSecretKey = "numendevsecret"
	bucketRegion    = "us-east-1"
)

func (c *Computer) ReadBlob(t *testing.T, vaultID, blobID []byte) []byte {
	t.Helper()
	address := c.getBlobURL(t, vaultID, blobID)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("read the blob: status %d, error %v", response.StatusCode, err)
	}
	return body
}

func (c *Computer) ReadStoredBlobs(t *testing.T) []byte {
	t.Helper()
	var stored []byte
	seen := map[string]bool{}
	for _, entry := range c.ReadCatalog(t) {
		if seen[string(entry.GetVaultId())] {
			continue
		}
		seen[string(entry.GetVaultId())] = true
		for _, head := range c.ReadChanges(t, entry.GetVaultId()) {
			if blobID := head.GetRevision().GetBlobId(); len(blobID) > 0 && !head.GetRevision().GetDeleted() {
				stored = append(stored, c.ReadBlob(t, entry.GetVaultId(), blobID)...)
			}
		}
	}
	return stored
}

func (c *Computer) DropBlob(t *testing.T, vaultID, blobID []byte) {
	t.Helper()
	address, err := url.Parse(c.getBlobURL(t, vaultID, blobID))
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, address.Scheme+"://"+address.Host+address.EscapedPath(), nil)
	if err != nil {
		t.Fatal(err)
	}
	signBucketRequest(request, GetNow().UTC())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
	if response.StatusCode >= 300 {
		t.Fatalf("delete the object of the blob: status %d", response.StatusCode)
	}
}

func (c *Computer) getBlobURL(t *testing.T, vaultID, blobID []byte) string {
	t.Helper()
	downloads, err := c.Cloud.GetBlobUrls(t.Context(), vaultID, [][]byte{blobID})
	if err != nil || len(downloads) != 1 || !downloads[0].GetFound() {
		t.Fatalf("get the address of the blob: %v, %v", downloads, err)
	}
	return downloads[0].GetUrl()
}

func signBucketRequest(r *http.Request, now time.Time) {
	stamp, day := now.Format("20060102T150405Z"), now.Format("20060102")
	payload := hex.EncodeToString(hashSHA256(nil))
	r.Header.Set("X-Amz-Date", stamp)
	r.Header.Set("X-Amz-Content-Sha256", payload)
	headers := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", r.URL.Host, payload, stamp)
	canonical := fmt.Sprintf("%s\n%s\n%s\n%s\nhost;x-amz-content-sha256;x-amz-date\n%s", r.Method, r.URL.EscapedPath(), r.URL.RawQuery, headers, payload)
	scope := day + "/" + bucketRegion + "/s3/aws4_request"
	toSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", stamp, scope, hex.EncodeToString(hashSHA256([]byte(canonical))))
	key := signHMAC([]byte("AWS4"+bucketSecretKey), day)
	for _, part := range []string{bucketRegion, "s3", "aws4_request"} {
		key = signHMAC(key, part)
	}
	signature := hex.EncodeToString(signHMAC(key, toSign))
	r.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=%s", bucketAccessKey, scope, signature))
}

func hashSHA256(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func signHMAC(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func (s *Server) ReadLastUploaded(t *testing.T) []byte {
	t.Helper()
	s.blob.mu.Lock()
	var key string
	for _, one := range s.blob.log {
		if one.Direction == "put" {
			key = one.Blob
		}
	}
	s.blob.mu.Unlock()
	if key == "" {
		t.Fatal("no body was uploaded")
	}
	bucket, err := url.Parse(s.Stack.BucketURL)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, bucket.Scheme+"://"+bucket.Host+"/"+key, nil)
	if err != nil {
		t.Fatal(err)
	}
	signBucketRequest(request, GetNow().UTC())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("read the object %s: status %d, error %v", key, response.StatusCode, err)
	}
	return body
}
