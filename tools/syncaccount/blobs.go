package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	amzDate        = "20060102T150405Z"
	amzDay         = "20060102"
	listPageLimit  = 1000
	objectLimit    = 64 << 20
	signedHeaders  = "host;x-amz-content-sha256;x-amz-date"
	emptyBodyHash  = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	serviceName    = "s3"
	signatureLabel = "AWS4-HMAC-SHA256"
)

type bucket struct {
	endpoint string
	name     string
	region   string
	access   string
	secret   string
	client   *http.Client
}

type objectPage struct {
	Truncated bool     `xml:"IsTruncated"`
	Next      string   `xml:"NextContinuationToken"`
	Contents  []object `xml:"Contents"`
}

type object struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
}

func runFindText(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("find-text", flag.ContinueOnError)
	endpoint := flags.String("bucket", "", "address of the S3 API of the bucket")
	name := flags.String("name", "numen-sync", "name of the bucket")
	region := flags.String("region", "us-east-1", "region of the bucket")
	access := flags.String("access-key", "numendev", "access key of the bucket")
	secret := flags.String("secret-key", "numendevsecret", "secret key of the bucket")
	text := flags.String("text", "", "the text the objects must not hold")
	since := flags.Int64("since", 0, "only objects written at this unix time or later are read")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *endpoint == "" || *text == "" {
		return errors.New("find-text needs -bucket and -text")
	}
	b := bucket{endpoint: strings.TrimSuffix(*endpoint, "/"), name: *name, region: *region, access: *access, secret: *secret, client: &http.Client{Timeout: requestTimeout}}
	read, err := b.findText(ctx, []byte(*text), time.Unix(*since, 0))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%d objects read, the text is in none\n", read)
	return err
}

func (b bucket) findText(ctx context.Context, text []byte, since time.Time) (int, error) {
	read, token := 0, ""
	for {
		page, err := b.list(ctx, token)
		if err != nil {
			return read, err
		}
		for _, one := range page.Contents {
			if one.LastModified.Before(since) {
				continue
			}
			body, err := b.get(ctx, one.Key)
			if err != nil {
				return read, err
			}
			read++
			if bytes.Contains(body, text) {
				return read, fmt.Errorf("the text is in the object %s", one.Key)
			}
		}
		if !page.Truncated {
			break
		}
		token = page.Next
	}
	if read == 0 {
		return 0, errors.New("the bucket holds no object written since then")
	}
	return read, nil
}

func (b bucket) list(ctx context.Context, token string) (objectPage, error) {
	query := url.Values{"list-type": {"2"}, "max-keys": {fmt.Sprint(listPageLimit)}}
	if token != "" {
		query.Set("continuation-token", token)
	}
	body, err := b.fetch(ctx, "/"+b.name, query)
	if err != nil {
		return objectPage{}, err
	}
	var page objectPage
	if err := xml.Unmarshal(body, &page); err != nil {
		return objectPage{}, fmt.Errorf("read the list of objects: %w", err)
	}
	return page, nil
}

func (b bucket) get(ctx context.Context, key string) ([]byte, error) {
	return b.fetch(ctx, "/"+b.name+"/"+key, nil)
}

func (b bucket) fetch(ctx context.Context, path string, query url.Values) ([]byte, error) {
	target, err := url.Parse(b.endpoint)
	if err != nil {
		return nil, err
	}
	target.Path = path
	target.RawQuery = encodeQuery(query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	b.sign(request, time.Now().UTC())
	response, err := b.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer closeQuietly(response.Body)
	body, err := io.ReadAll(io.LimitReader(response.Body, objectLimit))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the bucket answered %d to %s", response.StatusCode, path)
	}
	return body, nil
}

func encodeQuery(query url.Values) string {
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, strings.ReplaceAll(url.QueryEscape(key), "+", "%20")+"="+strings.ReplaceAll(url.QueryEscape(query.Get(key)), "+", "%20"))
	}
	return strings.Join(parts, "&")
}

func (b bucket) sign(request *http.Request, now time.Time) {
	stamp, day := now.Format(amzDate), now.Format(amzDay)
	request.Header.Set("X-Amz-Date", stamp)
	request.Header.Set("X-Amz-Content-Sha256", emptyBodyHash)
	canonical := strings.Join([]string{
		http.MethodGet, request.URL.EscapedPath(), request.URL.RawQuery,
		"host:" + request.URL.Host + "\nx-amz-content-sha256:" + emptyBodyHash + "\nx-amz-date:" + stamp + "\n",
		signedHeaders, emptyBodyHash,
	}, "\n")
	scope := strings.Join([]string{day, b.region, serviceName, "aws4_request"}, "/")
	hash := sha256.Sum256([]byte(canonical))
	toSign := strings.Join([]string{signatureLabel, stamp, scope, hex.EncodeToString(hash[:])}, "\n")
	key := signHmac([]byte("AWS4"+b.secret), day)
	for _, part := range []string{b.region, serviceName, "aws4_request"} {
		key = signHmac(key, part)
	}
	signature := hex.EncodeToString(signHmac(key, toSign))
	request.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", signatureLabel, b.access, scope, signedHeaders, signature))
}

func signHmac(key []byte, text string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(text))
	return mac.Sum(nil)
}
