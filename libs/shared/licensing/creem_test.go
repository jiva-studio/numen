package licensing_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/licensing"
)

func writeReply(t *testing.T, w http.ResponseWriter, statusCode int, body string) {
	t.Helper()
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func decodeBodyMap(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("decode request body: %v", err)
	}
	return body
}

func TestCreemBuildCheckoutURL(t *testing.T) {
	t.Parallel()
	p := licensing.NewCreemProvider()

	urlFromID, err := p.BuildCheckoutURL("prod_12345", "")
	if err != nil {
		t.Fatalf("BuildCheckoutURL(prod_12345) error = %v", err)
	}
	if urlFromID != "https://creem.io/payment/prod_12345" {
		t.Fatalf("BuildCheckoutURL(prod_12345) = %q, want https://creem.io/payment/prod_12345", urlFromID)
	}

	explicitURL, err := p.BuildCheckoutURL("prod_12345", "https://custom.example.com/pay/123")
	if err != nil {
		t.Fatalf("BuildCheckoutURL(explicit) error = %v", err)
	}
	if explicitURL != "https://custom.example.com/pay/123" {
		t.Fatalf("BuildCheckoutURL(explicit) = %q, want https://custom.example.com/pay/123", explicitURL)
	}

	if _, err := p.BuildCheckoutURL("", "http://insecure.example.com"); !errors.Is(err, licensing.ErrInvalidArgument) {
		t.Fatalf("BuildCheckoutURL(http) error = %v, want ErrInvalidArgument", err)
	}
	if _, err := p.BuildCheckoutURL("", ""); !errors.Is(err, licensing.ErrInvalidArgument) {
		t.Fatalf("BuildCheckoutURL(empty) error = %v, want ErrInvalidArgument", err)
	}
}

func TestCreemActivateValidateAndDeactivate(t *testing.T) {
	t.Parallel()
	var sawAPIKey atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerVal := r.Header.Get("X-Api-Key")
		sawAPIKey.Store(&headerVal)
		switch r.URL.Path {
		case "/v1/licenses/activate":
			body := decodeBodyMap(t, r)
			if body["key"] != "CREEM-KEY-1" || body["instance_name"] != "akd-laptop" {
				http.Error(w, `{"error":"Bad Request","message":["invalid payload"]}`, http.StatusBadRequest)
				return
			}
			writeReply(t, w, http.StatusOK, `{
				"id": "lic_1",
				"product_id": "prod_sync",
				"status": "active",
				"expires_at": "2027-01-15T12:00:00Z",
				"instance": {
					"id": "li_999",
					"name": "akd-laptop",
					"status": "active"
				}
			}`)
		case "/v1/licenses/validate":
			writeReply(t, w, http.StatusOK, `{
				"id": "lic_1",
				"product_id": "prod_sync",
				"status": "active",
				"expires_at": "2027-02-15T12:00:00Z",
				"instance": {
					"id": "li_999",
					"status": "active"
				}
			}`)
		case "/v1/licenses/deactivate":
			writeReply(t, w, http.StatusOK, `{"id":"lic_1","status":"inactive"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := licensing.NewCreemProvider()
	p.Configure(srv.URL, "creem_test_secret", srv.Client())

	if info := p.GetInfo(); info.ID != licensing.ProviderIDCreem {
		t.Fatalf("GetInfo().ID = %q, want %q", info.ID, licensing.ProviderIDCreem)
	}

	act, err := p.Activate(t.Context(), licensing.ActivationRequest{
		Key:          "  CREEM-KEY-1 \n",
		InstanceName: "akd-laptop",
		ProductID:    "prod_sync",
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if gotKey := sawAPIKey.Load(); gotKey == nil || *gotKey != "creem_test_secret" {
		t.Fatalf("x-api-key = %v, want creem_test_secret", gotKey)
	}
	if act.InstanceID != "li_999" || act.Status != licensing.StatusActive || act.ProductID != "prod_sync" {
		t.Fatalf("Activate() = %+v", act)
	}
	wantExp := time.Date(2027, 1, 15, 12, 0, 0, 0, time.UTC)
	if act.ExpiresAt == nil || !act.ExpiresAt.Equal(wantExp) {
		t.Fatalf("Activate().ExpiresAt = %v, want %v", act.ExpiresAt, wantExp)
	}

	val, err := p.Validate(t.Context(), "CREEM-KEY-1", "li_999")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if val.Status != licensing.StatusActive || val.ExpiresAt == nil {
		t.Fatalf("Validate() = %+v", val)
	}

	if err := p.Deactivate(t.Context(), "CREEM-KEY-1", "li_999"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
}

func TestCreemActivateRollsBackSlotOnProductMismatch(t *testing.T) {
	t.Parallel()
	var deactivatedInstance atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/licenses/activate":
			writeReply(t, w, http.StatusOK, `{
				"id": "lic_wrong",
				"product_id": "prod_other",
				"status": "active",
				"instance": {"id": "li_wrong_slot", "status": "active"}
			}`)
		case "/v1/licenses/deactivate":
			body := decodeBodyMap(t, r)
			inst := body["instance_id"]
			deactivatedInstance.Store(&inst)
			writeReply(t, w, http.StatusOK, `{"status":"inactive"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := licensing.NewCreemProvider()
	p.Configure(srv.URL, "", nil)
	_, err := p.Activate(t.Context(), licensing.ActivationRequest{
		Key:          "CREEM-KEY-OTHER",
		InstanceName: "akd-laptop",
		ProductID:    "prod_expected",
	})
	if !errors.Is(err, licensing.ErrProductMismatch) {
		t.Fatalf("Activate() error = %v, want ErrProductMismatch", err)
	}
	if gotID := deactivatedInstance.Load(); gotID == nil || *gotID != "li_wrong_slot" {
		t.Fatalf("deactivated instance = %v, want li_wrong_slot", gotID)
	}
}

func TestCreemErrorClassification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "not found 404",
			statusCode: http.StatusNotFound,
			body:       `{"error":"Not Found","message":["License not found"]}`,
			wantErr:    licensing.ErrLicenseNotFound,
		},
		{
			name:       "expired 410",
			statusCode: http.StatusGone,
			body:       `{"error":"Gone","message":["License has expired"]}`,
			wantErr:    licensing.ErrLicenseExpired,
		},
		{
			name:       "activation limit 403",
			statusCode: http.StatusForbidden,
			body:       `{"error":"Forbidden","message":["Activation limit reached"]}`,
			wantErr:    licensing.ErrActivationLimitReached,
		},
		{
			name:       "disabled 400",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"Bad Request","message":["License is disabled"]}`,
			wantErr:    licensing.ErrLicenseDisabled,
		},
		{
			name:       "rate limited 429",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":"Too Many Requests"}`,
			wantErr:    licensing.ErrProviderUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeReply(t, w, tc.statusCode, tc.body)
			}))
			defer srv.Close()

			p := licensing.NewCreemProvider()
			p.Configure(srv.URL, "", nil)
			_, err := p.Activate(t.Context(), licensing.ActivationRequest{
				Key:          "KEY",
				InstanceName: "machine",
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Activate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
