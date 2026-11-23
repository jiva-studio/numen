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
	defaultCreemAPIBase      = "https://api.creem.io"
	defaultCreemTestAPIBase  = "https://test-api.creem.io"
	defaultCreemCheckoutBase = "https://creem.io/payment/"
)

// CreemProvider manages machine license activations through the Creem.io API or its proxy.
type CreemProvider struct {
	baseURL string
	apiKey  string
	client  HTTPDoer
}

// NewCreemProvider returns a CreemProvider configured with the default Creem API base URL.
func NewCreemProvider() *CreemProvider {
	return &CreemProvider{
		baseURL: defaultCreemAPIBase,
		client:  getDefaultHTTPClient(),
	}
}

// Configure overrides the API base URL, API key and HTTP client when non-empty.
func (p *CreemProvider) Configure(baseURL, apiKey string, client HTTPDoer) {
	trimmedKey := strings.TrimSpace(apiKey)
	if trimmedKey != "" {
		p.apiKey = trimmedKey
	}
	trimmedURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedURL != "" {
		p.baseURL = trimmedURL
	} else if strings.HasPrefix(p.apiKey, "creem_test_") {
		p.baseURL = defaultCreemTestAPIBase
	}
	if client != nil {
		p.client = client
	}
}

// GetInfo returns the display metadata and payment methods for Creem.
func (p *CreemProvider) GetInfo() ProviderInfo {
	info, _ := GetProviderInfo(ProviderIDCreem)
	return info
}

func (p *CreemProvider) BuildCheckoutURL(productID, rawURL string) (string, error) {
	return buildCheckoutURLWithBase(defaultCreemCheckoutBase, productID, rawURL)
}

type creemActivatePayload struct {
	Key          string `json:"key"`
	InstanceName string `json:"instance_name"`
}

type creemInstancePayload struct {
	Key        string `json:"key"`
	InstanceID string `json:"instance_id"`
}

type creemLicenseResponse struct {
	ID        string                `json:"id"`
	ProductID string                `json:"product_id"`
	Status    string                `json:"status"`
	ExpiresAt string                `json:"expires_at"`
	Instance  *creemInstanceSection `json:"instance"`
}

type creemInstanceSection struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Activate registers a machine instance for a license key in Creem.
func (p *CreemProvider) Activate(ctx context.Context, req ActivationRequest) (ActivationResult, error) {
	key := NormalizeKey(req.Key)
	instanceName := strings.TrimSpace(req.InstanceName)
	if key == "" || instanceName == "" {
		return ActivationResult{}, fmt.Errorf("%w: key and instance name are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/v1/licenses/activate",
		p.getHeaders(),
		creemActivatePayload{Key: key, InstanceName: instanceName},
	)
	if err != nil {
		return ActivationResult{}, err
	}
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return ActivationResult{}, mapCreemHTTPError(code, raw)
	}
	var parsed creemLicenseResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ActivationResult{}, fmt.Errorf("%w: decode creem activation response: %w", ErrProviderUnavailable, err)
	}
	return p.evaluateCreemActivation(ctx, key, strings.TrimSpace(req.ProductID), parsed)
}

func (p *CreemProvider) evaluateCreemActivation(
	ctx context.Context,
	key, expectedProductID string,
	parsed creemLicenseResponse,
) (ActivationResult, error) {
	instanceID := ""
	if parsed.Instance != nil {
		instanceID = strings.TrimSpace(parsed.Instance.ID)
	}
	if expectedProductID != "" && parsed.ProductID != "" && parsed.ProductID != expectedProductID {
		mismatchErr := fmt.Errorf("%w: got %q, want %q", ErrProductMismatch, parsed.ProductID, expectedProductID)
		if instanceID != "" {
			return ActivationResult{}, errors.Join(mismatchErr, p.deactivateAfterMismatch(ctx, key, instanceID))
		}
		return ActivationResult{}, mismatchErr
	}
	status := resolveCreemStatus(parsed)
	if err := getStatusError(status); err != nil {
		return ActivationResult{}, err
	}
	if instanceID == "" {
		return ActivationResult{}, fmt.Errorf("%w: missing instance id in creem response", ErrProviderUnavailable)
	}
	expiresAt, err := parseOptionalTime(parsed.ExpiresAt)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("%w: invalid expires_at in creem response: %w", ErrProviderUnavailable, err)
	}
	return ActivationResult{
		InstanceID: instanceID,
		ProductID:  parsed.ProductID,
		Status:     status,
		ExpiresAt:  expiresAt,
	}, nil
}

func (p *CreemProvider) deactivateAfterMismatch(ctx context.Context, key, instanceID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return p.Deactivate(cleanupCtx, key, instanceID)
}

// Validate checks whether a license key and machine instance remain active in Creem.
func (p *CreemProvider) Validate(ctx context.Context, key, instanceID string) (ValidationResult, error) {
	cleanKey := NormalizeKey(key)
	cleanInstance := strings.TrimSpace(instanceID)
	if cleanKey == "" || cleanInstance == "" {
		return ValidationResult{}, fmt.Errorf("%w: key and instance id are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/v1/licenses/validate",
		p.getHeaders(),
		creemInstancePayload{Key: cleanKey, InstanceID: cleanInstance},
	)
	if err != nil {
		return ValidationResult{}, err
	}
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return ValidationResult{}, mapCreemHTTPError(code, raw)
	}
	var parsed creemLicenseResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ValidationResult{}, fmt.Errorf("%w: decode creem validation response: %w", ErrProviderUnavailable, err)
	}
	expiresAt, err := parseOptionalTime(parsed.ExpiresAt)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("%w: invalid expires_at in creem response: %w", ErrProviderUnavailable, err)
	}
	return ValidationResult{
		Status:    resolveCreemStatus(parsed),
		ExpiresAt: expiresAt,
	}, nil
}

// Deactivate releases a machine instance slot for a license key in Creem.
func (p *CreemProvider) Deactivate(ctx context.Context, key, instanceID string) error {
	cleanKey := NormalizeKey(key)
	cleanInstance := strings.TrimSpace(instanceID)
	if cleanKey == "" || cleanInstance == "" {
		return fmt.Errorf("%w: key and instance id are required", ErrInvalidArgument)
	}
	code, raw, err := sendJSON(
		ctx,
		p.client,
		p.baseURL+"/v1/licenses/deactivate",
		p.getHeaders(),
		creemInstancePayload{Key: cleanKey, InstanceID: cleanInstance},
	)
	if err != nil {
		return err
	}
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return mapCreemHTTPError(code, raw)
}

func (p *CreemProvider) getHeaders() map[string]string {
	if p.apiKey == "" {
		return nil
	}
	return map[string]string{"x-api-key": p.apiKey}
}

func resolveCreemStatus(parsed creemLicenseResponse) Status {
	status := mapStatus(parsed.Status)
	if status != StatusActive {
		return status
	}
	if parsed.Instance != nil && parsed.Instance.Status != "" {
		instStatus := mapStatus(parsed.Instance.Status)
		if instStatus != StatusActive {
			return instStatus
		}
	}
	return StatusActive
}

func mapCreemHTTPError(code int, raw []byte) error {
	msg := extractCreemErrorMessage(raw)
	lower := strings.ToLower(msg)
	switch code {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrLicenseNotFound, msg)
	case http.StatusGone:
		return fmt.Errorf("%w: %s", ErrLicenseExpired, msg)
	case http.StatusUnprocessableEntity:
		return classifyCreemMessage(lower, msg, ErrActivationLimitReached)
	case http.StatusForbidden:
		if strings.Contains(lower, "limit") || strings.Contains(lower, "cap") || strings.Contains(lower, "activation") {
			return fmt.Errorf("%w: %s", ErrActivationLimitReached, msg)
		}
		return classifyCreemMessage(lower, msg, ErrLicenseDisabled)
	case http.StatusBadRequest:
		return classifyCreemMessage(lower, msg, ErrLicenseDisabled)
	default:
		return fmt.Errorf("%w: http %d: %s", ErrProviderUnavailable, code, msg)
	}
}

func classifyCreemMessage(lower, msg string, fallback error) error {
	switch {
	case strings.Contains(lower, "limit") || strings.Contains(lower, "max"):
		return fmt.Errorf("%w: %s", ErrActivationLimitReached, msg)
	case strings.Contains(lower, "expir"):
		return fmt.Errorf("%w: %s", ErrLicenseExpired, msg)
	case strings.Contains(lower, "not found"):
		return fmt.Errorf("%w: %s", ErrLicenseNotFound, msg)
	default:
		return fmt.Errorf("%w: %s", fallback, msg)
	}
}

func extractCreemErrorMessage(raw []byte) string {
	if msg := parseCreemErrorJSON(raw); msg != "" {
		return msg
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "request failed"
	}
	return trimmed
}

func parseCreemErrorJSON(raw []byte) string {
	var body struct {
		Error   string          `json:"error"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	var list []string
	if err := json.Unmarshal(body.Message, &list); err == nil && len(list) > 0 {
		return strings.Join(list, "; ")
	}
	var single string
	if err := json.Unmarshal(body.Message, &single); err == nil && single != "" {
		return single
	}
	return strings.TrimSpace(body.Error)
}
