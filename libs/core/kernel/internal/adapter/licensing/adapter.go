// Package licensing adapts the shared licensing provider clients and the machine's identity for the
// license use cases.
package licensing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	sharedlicensing "github.com/jiva-studio/numen/libs/shared/licensing"
)

const (
	envCreemBaseURL = "NUMEN_CREEM_BASE_URL"
	envCreemAPIKey  = "NUMEN_CREEM_API_KEY"
	envDodoBaseURL  = "NUMEN_DODO_BASE_URL"
	fingerprintSalt = "numen-license-v1:"
)

// Adapter wraps the Creem and Dodo licensing providers and reads the machine fingerprint and
// instance name.
type Adapter struct {
	providers map[string]provider

	mu          sync.Mutex
	subscribers map[int]chan struct{}
	nextSubID   int
	handleChange    func(ctx context.Context)
}

// NewAdapter creates a licensing adapter configured from the environment.
func NewAdapter(handleChange func(ctx context.Context)) *Adapter {
	creem := sharedlicensing.NewCreemProvider()
	creem.Configure(os.Getenv(envCreemBaseURL), os.Getenv(envCreemAPIKey), nil)

	dodo := sharedlicensing.NewDodoProvider()
	dodo.Configure(os.Getenv(envDodoBaseURL), nil)

	return &Adapter{
		providers: map[string]provider{
			sharedlicensing.ProviderIDCreem: creem,
			sharedlicensing.ProviderIDDodo:  dodo,
		},
		subscribers: make(map[int]chan struct{}),
		handleChange:    handleChange,
	}
}

// ListProviderOptions resolves the display names, checkout URLs and payment methods for the
// providers a license declares.
func (a *Adapter) ListProviderOptions(declared []manifest.LicenseProvider) []license.ProviderOption {
	var options []license.ProviderOption
	for _, one := range declared {
		if !sharedlicensing.IsKnownProvider(one.Provider) {
			continue
		}
		prov, ok := a.providers[one.Provider]
		if !ok {
			continue
		}
		info := prov.GetInfo()
		checkoutURL, err := prov.BuildCheckoutURL(one.ProductID, one.URL)
		if err != nil {
			continue
		}
		options = append(options, license.ProviderOption{
			Provider:       info.ID,
			ProviderName:   info.Name,
			ProductID:      one.ProductID,
			CheckoutURL:    checkoutURL,
			PaymentMethods: info.PaymentMethods,
		})
	}
	return options
}

// Activate tries each declared provider in order and returns the activation result or the most
// specific failure.
func (a *Adapter) Activate(
	ctx context.Context,
	declared []manifest.LicenseProvider,
	key, instanceName string,
) (license.ActivationResult, error) {
	cleanedKey := sharedlicensing.NormalizeKey(key)
	_ = sharedlicensing.GetKeyHint(cleanedKey)
	var errs []error
	for _, one := range declared {
		prov, ok := a.providers[one.Provider]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: %w", one.Provider, license.ErrProviderUnavailable))
			continue
		}
		expectedProductID := resolveExpectedProductID(one)
		if one.Provider == sharedlicensing.ProviderIDDodo && expectedProductID == "" {
			errs = append(errs, fmt.Errorf("%s: %w: product ID is required", prov.GetInfo().Name, license.ErrInvalidKey))
			continue
		}
		res, err := prov.Activate(ctx, sharedlicensing.ActivationRequest{
			Key:          cleanedKey,
			InstanceName: instanceName,
			ProductID:    expectedProductID,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", prov.GetInfo().Name, err))
			continue
		}
		return license.ActivationResult{
			Provider:   one.Provider,
			InstanceID: res.InstanceID,
			Status:     mapSharedStatus(res.Status),
			ExpiresAt:  res.ExpiresAt,
		}, nil
	}
	chosen := sharedlicensing.SelectError(errs)
	if chosen == nil {
		return license.ActivationResult{}, license.ErrNotFound
	}
	return license.ActivationResult{}, mapSharedError(chosen)
}

// Validate checks whether a license key and machine instance remain valid at the named provider.
func (a *Adapter) Validate(
	ctx context.Context,
	provider, key, instanceID string,
) (license.ValidationResult, error) {
	prov, ok := a.providers[provider]
	if !ok {
		return license.ValidationResult{}, fmt.Errorf("%w: unknown provider %s", license.ErrProviderUnavailable, provider)
	}
	res, err := prov.Validate(ctx, key, instanceID)
	if err != nil {
		return license.ValidationResult{}, mapSharedError(err)
	}
	return license.ValidationResult{
		Status:    mapSharedStatus(res.Status),
		ExpiresAt: res.ExpiresAt,
	}, nil
}

// Deactivate releases a machine instance slot at the named provider.
func (a *Adapter) Deactivate(ctx context.Context, provider, key, instanceID string) error {
	prov, ok := a.providers[provider]
	if !ok {
		return fmt.Errorf("%w: unknown provider %s", license.ErrProviderUnavailable, provider)
	}
	if err := prov.Deactivate(ctx, key, instanceID); err != nil {
		return mapSharedError(err)
	}
	return nil
}

// GetMachineFingerprint returns the deterministic SHA-256 fingerprint of this machine.
func (a *Adapter) GetMachineFingerprint() string {
	rawID := readMachineID()
	sum := sha256.Sum256([]byte(fingerprintSalt + rawID))
	return hex.EncodeToString(sum[:])
}

// GetInstanceName returns the human-readable hostname of this machine.
func (a *Adapter) GetInstanceName() string {
	if host, err := os.Hostname(); err == nil {
		if trimmed := strings.TrimSpace(host); trimmed != "" {
			return trimmed
		}
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}

// NotifyChange broadcasts a change signal to all subscribers and runs the host callback.
func (a *Adapter) NotifyChange(ctx context.Context) {
	a.mu.Lock()
	for _, ch := range a.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	a.mu.Unlock()
	if a.handleChange != nil {
		a.handleChange(ctx)
	}
}

// Subscribe registers a listener channel for license changes.
func (a *Adapter) Subscribe() (<-chan struct{}, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := a.nextSubID
	a.nextSubID++
	ch := make(chan struct{}, 1)
	a.subscribers[id] = ch
	return ch, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		delete(a.subscribers, id)
	}
}

func resolveExpectedProductID(decl manifest.LicenseProvider) string {
	if trimmed := strings.TrimSpace(decl.ProductID); trimmed != "" {
		return trimmed
	}
	if decl.Provider == sharedlicensing.ProviderIDDodo {
		return getCheckoutProductID(decl.URL, "/buy/")
	}
	return getCheckoutProductID(decl.URL, "/payment/")
}

func getCheckoutProductID(rawURL, marker string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	idx := strings.LastIndex(parsed.Path, marker)
	if idx < 0 {
		return ""
	}
	rest := parsed.Path[idx+len(marker):]
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		rest = rest[:slash]
	}
	return strings.TrimSpace(rest)
}

func readMachineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if data, err := os.ReadFile(path); err == nil {
			if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
				return trimmed
			}
		}
	}
	if machineID := readPlatformMachineID(); machineID != "" {
		return machineID
	}
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	return strings.TrimSpace(host) + ":" + runtime.GOOS + ":" + runtime.GOARCH
}

func readPlatformMachineID() string {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(context.Background(), "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	case "windows":
		command = exec.CommandContext(context.Background(), "wmic", "csproduct", "get", "uuid")
	default:
		return ""
	}
	output, err := command.Output()
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if runtime.GOOS == "darwin" && strings.Contains(trimmed, "IOPlatformUUID") {
			parts := strings.Split(trimmed, "=")
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`)
			}
		}
		if runtime.GOOS == "windows" && trimmed != "" && !strings.EqualFold(trimmed, "UUID") {
			return trimmed
		}
	}
	return ""
}

func mapSharedStatus(s sharedlicensing.Status) license.Status {
	if s.IsActive() {
		return license.StatusActive
	}
	switch s {
	case sharedlicensing.StatusActive:
		return license.StatusActive
	case sharedlicensing.StatusExpired:
		return license.StatusExpired
	case sharedlicensing.StatusDisabled:
		return license.StatusDisabled
	case sharedlicensing.StatusInactive:
		return license.StatusInactive
	}
	return license.StatusInactive
}

func mapSharedError(err error) error {
	switch {
	case errors.Is(err, sharedlicensing.ErrActivationLimitReached):
		return fmt.Errorf("%w: %s", license.ErrActivationLimitReached, err.Error())
	case errors.Is(err, sharedlicensing.ErrLicenseExpired):
		return fmt.Errorf("%w: %s", license.ErrExpired, err.Error())
	case errors.Is(err, sharedlicensing.ErrLicenseDisabled):
		return fmt.Errorf("%w: %s", license.ErrDisabled, err.Error())
	case errors.Is(err, sharedlicensing.ErrProductMismatch):
		return fmt.Errorf("%w: %s", license.ErrProductMismatch, err.Error())
	case errors.Is(err, sharedlicensing.ErrProviderUnavailable):
		return fmt.Errorf("%w: %s", license.ErrProviderUnavailable, err.Error())
	case errors.Is(err, sharedlicensing.ErrLicenseNotFound):
		return fmt.Errorf("%w: %s", license.ErrNotFound, err.Error())
	case errors.Is(err, sharedlicensing.ErrInvalidArgument):
		return fmt.Errorf("%w: %s", license.ErrInvalidKey, err.Error())
	default:
		return fmt.Errorf("%w: %s", license.ErrProviderUnavailable, err.Error())
	}
}

// provider activates, validates and deactivates machine license instances for one service.
type provider interface {
	GetInfo() sharedlicensing.ProviderInfo
	BuildCheckoutURL(productID, rawURL string) (string, error)
	Activate(ctx context.Context, req sharedlicensing.ActivationRequest) (sharedlicensing.ActivationResult, error)
	Validate(ctx context.Context, key, instanceID string) (sharedlicensing.ValidationResult, error)
	Deactivate(ctx context.Context, key, instanceID string) error
}
