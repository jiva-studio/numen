package licensing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDodoAPIBase      = "https://live.dodopayments.com"
	defaultDodoCheckoutBase = "https://checkout.dodopayments.com/buy/"
)

// DodoProvider manages machine license activations through Dodo Payments' public license endpoints.
type DodoProvider struct {
	baseURL string
	client  HTTPDoer
}

// NewDodoProvider returns a DodoProvider configured with the default Dodo Payments API base URL.
func NewDodoProvider() *DodoProvider {
	return &DodoProvider{
		baseURL: defaultDodoAPIBase,
		client:  getDefaultHTTPClient(),
	}
}

// Configure overrides the API base URL and HTTP client when non-empty.
func (p *DodoProvider) Configure(baseURL string, client HTTPDoer) {
	trimmedURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedURL != "" {
		p.baseURL = trimmedURL
	}
	if client != nil {
		p.client = client
	}
}

// GetInfo returns the display metadata and payment methods for Dodo Payments.
func (p *DodoProvider) GetInfo() ProviderInfo {
	info, _ := GetProviderInfo(ProviderIDDodo)
	return info
}

// BuildCheckoutURL returns the explicit checkout URL or builds one from the Dodo Payments product ID.
func (p *DodoProvider) BuildCheckoutURL(productID, rawURL string) (string, error) {
	return buildCheckoutURLWithBase(defaultDodoCheckoutBase, productID, rawURL)
}

type dodoActivatePayload struct {
	LicenseKey string `json:"license_key"`
	Name       string `json:"name"`
}

type dodoInstancePayload struct {
	LicenseKey           string `json:"license_key"`
	LicenseKeyInstanceID string `json:"license_key_instance_id"`
}

type dodoActivateResponse struct {
	ID        string             `json:"id"`
	ExpiresAt string             `json:"expires_at"`
	Product   dodoProductSection `json:"product"`
}

type dodoProductSection struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
}

type dodoValidateResponse struct {
	Valid     bool   `json:"valid"`
	ExpiresAt string `json:"expires_at"`
}

// Activate registers a machine instance for a license key in Dodo Payments.
func (p *DodoProvider) Activate(ctx context.Context, req ActivationRequest) (ActivationResult, error) {
	key := NormalizeKey(req.Key)
	instanceName := strings.TrimSpace(req.InstanceName)
	if key == "" || instanceName == "" {
		return ActivationResult{}, fmt.Errorf("%w: key and instance name are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/licenses/activate",
		nil,
		dodoActivatePayload{LicenseKey: key, Name: instanceName},
	)
	if err != nil {
		return ActivationResult{}, err
	}
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return ActivationResult{}, mapDodoHTTPError(code, raw)
	}
	var parsed dodoActivateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ActivationResult{}, fmt.Errorf("%w: decode dodo activation response: %w", ErrProviderUnavailable, err)
	}
	return p.evaluateDodoActivation(ctx, key, strings.TrimSpace(req.ProductID), parsed)
}

func (p *DodoProvider) evaluateDodoActivation(
	ctx context.Context,
	key, expectedProductID string,
	parsed dodoActivateResponse,
) (ActivationResult, error) {
	instanceID := strings.TrimSpace(parsed.ID)
	actualProductID := strings.TrimSpace(parsed.Product.ProductID)
	if expectedProductID != "" && actualProductID != "" && actualProductID != expectedProductID {
		mismatchErr := fmt.Errorf("%w: got %q, want %q", ErrProductMismatch, actualProductID, expectedProductID)
		if instanceID != "" {
			return ActivationResult{}, errors.Join(mismatchErr, p.deactivateAfterMismatch(ctx, key, instanceID))
		}
		return ActivationResult{}, mismatchErr
	}
	if instanceID == "" {
		return ActivationResult{}, fmt.Errorf("%w: missing instance id in dodo response", ErrProviderUnavailable)
	}
	expiresAt, err := parseOptionalTime(parsed.ExpiresAt)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("%w: invalid expires_at in dodo response: %w", ErrProviderUnavailable, err)
	}
	return ActivationResult{
		InstanceID: instanceID,
		ProductID:  actualProductID,
		Status:     StatusActive,
		ExpiresAt:  expiresAt,
	}, nil
}

func (p *DodoProvider) deactivateAfterMismatch(ctx context.Context, key, instanceID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return p.Deactivate(cleanupCtx, key, instanceID)
}

// Validate checks whether a license key and machine instance remain valid in Dodo Payments.
func (p *DodoProvider) Validate(ctx context.Context, key, instanceID string) (ValidationResult, error) {
	cleanKey := NormalizeKey(key)
	cleanInstance := strings.TrimSpace(instanceID)
	if cleanKey == "" || cleanInstance == "" {
		return ValidationResult{}, fmt.Errorf("%w: key and instance id are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/licenses/validate",
		nil,
		dodoInstancePayload{LicenseKey: cleanKey, LicenseKeyInstanceID: cleanInstance},
	)
	if err != nil {
		return ValidationResult{}, err
	}
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return ValidationResult{}, mapDodoHTTPError(code, raw)
	}
	var parsed dodoValidateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ValidationResult{}, fmt.Errorf("%w: decode dodo validation response: %w", ErrProviderUnavailable, err)
	}
	expiresAt, err := parseOptionalTime(parsed.ExpiresAt)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("%w: invalid expires_at in dodo response: %w", ErrProviderUnavailable, err)
	}
	status := StatusInactive
	if parsed.Valid {
		status = StatusActive
	}
	return ValidationResult{
		Status:    status,
		ExpiresAt: expiresAt,
	}, nil
}

// Deactivate releases a machine instance slot for a license key in Dodo Payments.
func (p *DodoProvider) Deactivate(ctx context.Context, key, instanceID string) error {
	cleanKey := NormalizeKey(key)
	cleanInstance := strings.TrimSpace(instanceID)
	if cleanKey == "" || cleanInstance == "" {
		return fmt.Errorf("%w: key and instance id are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/licenses/deactivate",
		nil,
		dodoInstancePayload{LicenseKey: cleanKey, LicenseKeyInstanceID: cleanInstance},
	)
	if err != nil {
		return err
	}
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return mapDodoHTTPError(code, raw)
}

func mapDodoHTTPError(code int, raw []byte) error {
	errCode, msg := extractDodoError(raw)
	upperCode := strings.ToUpper(errCode)
	lowerMsg := strings.ToLower(msg)
	switch {
	case upperCode == "LICENSE_KEY_LIMIT_REACHED" || code == http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", ErrActivationLimitReached, msg)
	case upperCode == "LICENSE_KEY_EXPIRED" || code == http.StatusGone || strings.Contains(lowerMsg, "expir"):
		return fmt.Errorf("%w: %s", ErrLicenseExpired, msg)
	case upperCode == "LICENSE_KEY_NOT_ACTIVE" || code == http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrLicenseDisabled, msg)
	case upperCode == "LICENSE_KEY_NOT_FOUND" || code == http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrLicenseNotFound, msg)
	case code == http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrLicenseDisabled, msg)
	default:
		return fmt.Errorf("%w: http %d: %s", ErrProviderUnavailable, code, msg)
	}
}

func extractDodoError(raw []byte) (string, string) {
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &body); err == nil {
		msg := strings.TrimSpace(body.Message)
		if msg == "" {
			msg = strings.TrimSpace(body.Code)
		}
		if msg != "" || body.Code != "" {
			return strings.TrimSpace(body.Code), msg
		}
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", "request failed"
	}
	return "", trimmed
}
