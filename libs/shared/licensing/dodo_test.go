package licensing_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/licensing"
)

func TestDodoBuildCheckoutURL(t *testing.T) {
	t.Parallel()
	p := licensing.NewDodoProvider()

	urlFromID, err := p.BuildCheckoutURL("pdt_supporter", "")
	if err != nil {
		t.Fatalf("BuildCheckoutURL(pdt_supporter) error = %v", err)
	}
	if urlFromID != "https://checkout.dodopayments.com/buy/pdt_supporter" {
		t.Fatalf("BuildCheckoutURL(pdt_supporter) = %q, want https://checkout.dodopayments.com/buy/pdt_supporter", urlFromID)
	}
}

func TestDodoActivateValidateAndDeactivate(t *testing.T) {
	t.Parallel()
	var validState atomic.Bool
	validState.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/licenses/activate":
			body := decodeBodyMap(t, r)
			if body["license_key"] != "DODO-KEY-1" || body["name"] != "akd-workstation" {
				http.Error(w, `{"code":"BAD_REQUEST","message":"invalid payload"}`, http.StatusBadRequest)
				return
			}
			writeReply(t, w, http.StatusCreated, `{
				"id": "lki_777",
				"license_key_id": "lic_777",
				"name": "akd-workstation",
				"product": {
					"product_id": "pdt_supporter",
					"name": "Numen Supporter"
				}
			}`)
		case "/licenses/validate":
			body := decodeBodyMap(t, r)
			if body["license_key"] != "DODO-KEY-1" || body["license_key_instance_id"] != "lki_777" {
				http.Error(w, `{"code":"LICENSE_KEY_NOT_FOUND","message":"not found"}`, http.StatusNotFound)
				return
			}
			if validState.Load() {
				writeReply(t, w, http.StatusOK, `{"valid":true}`)
			} else {
				writeReply(t, w, http.StatusOK, `{"valid":false}`)
			}
		case "/licenses/deactivate":
			writeReply(t, w, http.StatusOK, `{"message":"License key instance deactivated successfully."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := licensing.NewDodoProvider()
	p.Configure(srv.URL, srv.Client())

	if info := p.GetInfo(); info.ID != licensing.ProviderIDDodo {
		t.Fatalf("GetInfo().ID = %q, want %q", info.ID, licensing.ProviderIDDodo)
	}

	act, err := p.Activate(t.Context(), licensing.ActivationRequest{
		Key:          "DODO-KEY-1",
		InstanceName: "akd-workstation",
		ProductID:    "pdt_supporter",
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if act.InstanceID != "lki_777" || act.ProductID != "pdt_supporter" || act.Status != licensing.StatusActive {
		t.Fatalf("Activate() = %+v", act)
	}

	val, err := p.Validate(t.Context(), "DODO-KEY-1", "lki_777")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if val.Status != licensing.StatusActive {
		t.Fatalf("Validate().Status = %q, want active", val.Status)
	}

	validState.Store(false)
	valInactive, err := p.Validate(t.Context(), "DODO-KEY-1", "lki_777")
	if err != nil {
		t.Fatalf("Validate(inactive) error = %v", err)
	}
	if valInactive.Status != licensing.StatusInactive {
		t.Fatalf("Validate(inactive).Status = %q, want inactive", valInactive.Status)
	}

	if err := p.Deactivate(t.Context(), "DODO-KEY-1", "lki_777"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
}

func TestDodoActivateRollsBackSlotOnProductMismatch(t *testing.T) {
	t.Parallel()
	var deactivatedInstance atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/licenses/activate":
			writeReply(t, w, http.StatusCreated, `{
				"id": "lki_wrong_slot",
				"product": {"product_id": "pdt_other", "name": "Other"}
			}`)
		case "/licenses/deactivate":
			body := decodeBodyMap(t, r)
			inst := body["license_key_instance_id"]
			deactivatedInstance.Store(&inst)
			writeReply(t, w, http.StatusOK, `{"message":"ok"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := licensing.NewDodoProvider()
	p.Configure(srv.URL, nil)
	_, err := p.Activate(t.Context(), licensing.ActivationRequest{
		Key:          "DODO-KEY-OTHER",
		InstanceName: "akd-workstation",
		ProductID:    "pdt_expected",
	})
	if !errors.Is(err, licensing.ErrProductMismatch) {
		t.Fatalf("Activate() error = %v, want ErrProductMismatch", err)
	}
	if gotID := deactivatedInstance.Load(); gotID == nil || *gotID != "lki_wrong_slot" {
		t.Fatalf("deactivated instance = %v, want lki_wrong_slot", gotID)
	}
}

func TestDodoErrorClassification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "limit reached 422",
			statusCode: http.StatusUnprocessableEntity,
			body:       `{"code":"LICENSE_KEY_LIMIT_REACHED","message":"License key activation limit reached"}`,
			wantErr:    licensing.ErrActivationLimitReached,
		},
		{
			name:       "not active 403",
			statusCode: http.StatusForbidden,
			body:       `{"code":"LICENSE_KEY_NOT_ACTIVE","message":"License key is not active"}`,
			wantErr:    licensing.ErrLicenseDisabled,
		},
		{
			name:       "not found 404",
			statusCode: http.StatusNotFound,
			body:       `{"code":"LICENSE_KEY_NOT_FOUND","message":"License key not found"}`,
			wantErr:    licensing.ErrLicenseNotFound,
		},
		{
			name:       "server error 503",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"message":"temporarily unavailable"}`,
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

			p := licensing.NewDodoProvider()
			p.Configure(srv.URL, nil)
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
