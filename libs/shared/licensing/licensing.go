package licensing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// Status is the state of a license key or machine activation reported by a provider.
type Status string

const (
	StatusActive   Status = "active"
	StatusExpired  Status = "expired"
	StatusInactive Status = "inactive"
	StatusDisabled Status = "disabled"
)

// IsActive reports whether the status grants access to the licensed feature.
func (s Status) IsActive() bool {
	return s == StatusActive
}

const (
	// ProviderIDCreem identifies the Creem.io licensing provider.
	ProviderIDCreem = "creem"
	// ProviderIDDodo identifies the Dodo Payments licensing provider.
	ProviderIDDodo = "dodo"
)

var (
	// ErrLicenseNotFound means the provider does not know the license key or instance ID.
	ErrLicenseNotFound = errors.New("license key not found")
	// ErrActivationLimitReached means the license key has used all of its machine activation slots.
	ErrActivationLimitReached = errors.New("license activation limit reached")
	// ErrLicenseExpired means the license key or subscription period has expired.
	ErrLicenseExpired = errors.New("license key expired")
	// ErrLicenseDisabled means the license key or instance is disabled, revoked or inactive.
	ErrLicenseDisabled = errors.New("license key is not active")
	// ErrProductMismatch means the activated license key belongs to a different product ID.
	ErrProductMismatch = errors.New("license key belongs to another product")
	// ErrProviderUnavailable means the provider could not be reached or answered with a transient failure.
	ErrProviderUnavailable = errors.New("license provider unavailable")
	// ErrInvalidArgument means a required input such as the license key, instance ID or checkout target is empty or invalid.
	ErrInvalidArgument = errors.New("invalid license argument")
	// ErrUnknownProvider means the provider identifier is not registered.
	ErrUnknownProvider = errors.New("unknown license provider")
)

// ProviderInfo describes a licensing provider and the payment methods its checkout supports.
type ProviderInfo struct {
	ID             string
	Name           string
	PaymentMethods []string
}

// ActivationRequest is the input for activating a license key on a machine.
type ActivationRequest struct {
	Key          string
	InstanceName string
	ProductID    string
}

// ActivationResult is the machine instance and license state returned when activation succeeds.
type ActivationResult struct {
	InstanceID string
	ProductID  string
	Status     Status
	ExpiresAt  *time.Time
}

// ValidationResult is the license state returned when validating an active machine instance.
type ValidationResult struct {
	Status    Status
	ExpiresAt *time.Time
}

// Provider activates, validates and deactivates machine license instances for one service.
type Provider interface {
	GetInfo() ProviderInfo
	BuildCheckoutURL(productID, rawURL string) (string, error)
	Activate(ctx context.Context, req ActivationRequest) (ActivationResult, error)
	Validate(ctx context.Context, key, instanceID string) (ValidationResult, error)
	Deactivate(ctx context.Context, key, instanceID string) error
}

// HTTPDoer executes an HTTP request.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// IsKnownProvider reports whether id names a supported licensing provider.
func IsKnownProvider(id string) bool {
	_, ok := GetProviderInfo(id)
	return ok
}

// GetProviderInfo returns the display name and payment methods of a known provider.
func GetProviderInfo(id string) (ProviderInfo, bool) {
	switch id {
	case ProviderIDCreem:
		return ProviderInfo{
			ID:             ProviderIDCreem,
			Name:           "Creem",
			PaymentMethods: []string{"Cards", "Apple Pay", "Google Pay", "Alipay"},
		}, true
	case ProviderIDDodo:
		return ProviderInfo{
			ID:             ProviderIDDodo,
			Name:           "Dodo Payments",
			PaymentMethods: []string{"Cards", "Apple Pay", "Google Pay", "Crypto (USDC)"},
		}, true
	default:
		return ProviderInfo{}, false
	}
}

// NormalizeKey trims surrounding whitespace and invisible format characters from a user-entered license key.
func NormalizeKey(key string) string {
	return strings.TrimFunc(key, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r)
	})
}

const hintVisibleRunes = 4

// GetKeyHint returns a masked representation of a license key showing at most the last four characters.
func GetKeyHint(key string) string {
	cleaned := NormalizeKey(key)
	if cleaned == "" {
		return ""
	}
	runes := []rune(cleaned)
	if len(runes) <= hintVisibleRunes {
		return "••••"
	}
	return "••••-" + string(runes[len(runes)-hintVisibleRunes:])
}

// SelectError chooses the most specific error from a list of provider activation or validation failures.
func SelectError(errs []error) error {
	var best error
	bestRank := 0
	for _, err := range errs {
		if err == nil {
			continue
		}
		rank := getErrorRank(err)
		if rank > bestRank {
			bestRank = rank
			best = err
		}
	}
	return best
}

const (
	rankUnknown = iota + 1
	rankInvalidArgument
	rankNotFound
	rankProviderUnavailable
	rankProductMismatch
	rankDisabled
	rankExpired
	rankLimitReached
)

func getErrorRank(err error) int {
	switch {
	case errors.Is(err, ErrActivationLimitReached):
		return rankLimitReached
	case errors.Is(err, ErrLicenseExpired):
		return rankExpired
	case errors.Is(err, ErrLicenseDisabled):
		return rankDisabled
	case errors.Is(err, ErrProductMismatch):
		return rankProductMismatch
	case errors.Is(err, ErrProviderUnavailable):
		return rankProviderUnavailable
	case errors.Is(err, ErrLicenseNotFound):
		return rankNotFound
	case errors.Is(err, ErrInvalidArgument):
		return rankInvalidArgument
	default:
		return rankUnknown
	}
}

func buildCheckoutURLWithBase(basePrefix, productID, rawURL string) (string, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL != "" {
		parsed, err := url.Parse(trimmedURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return "", fmt.Errorf("%w: checkout url must be a valid https address", ErrInvalidArgument)
		}
		return trimmedURL, nil
	}
	trimmedID := strings.TrimSpace(productID)
	if trimmedID == "" {
		return "", fmt.Errorf("%w: product id or checkout url is required", ErrInvalidArgument)
	}
	if strings.ContainsAny(trimmedID, "/?#\\ ") {
		return "", fmt.Errorf("%w: product id %q contains invalid characters", ErrInvalidArgument, trimmedID)
	}
	return basePrefix + url.PathEscape(trimmedID), nil
}
