// Package license holds what a license declared by one or more plugins is, how its stored envelope
// in the key store is decoded and encoded, and how its status is evaluated at an instant in time.
package license

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Status is the state of a license on this machine.
type Status string

const (
	// StatusInactive is a license with no key activated on this machine.
	StatusInactive Status = "inactive"
	// StatusActive is a license activated on this machine and verified within its grace window.
	StatusActive Status = "active"
	// StatusExpired is a license whose subscription or validity period has ended.
	StatusExpired Status = "expired"
	// StatusUnverified is a stored key that could not be verified within the offline grace window or
	// after a clock rollback, or whose provider disabled it.
	StatusUnverified Status = "unverified"
	// StatusDisabled is the status stored when a provider reports a key disabled or revoked.
	StatusDisabled Status = "disabled"
)

const (
	// ProviderCustom is the provider recorded in an envelope when the license declares no external
	// licensing providers.
	ProviderCustom = "custom"

	// ClockRollbackTolerance is the largest backward jump in UTC clock time tolerated before offline
	// grace is suspended until online verification succeeds.
	ClockRollbackTolerance = 48 * time.Hour

	// BillingDunningGrace is the offline dunning window after ExpiresAt before a license without a
	// fresh online check turns expired.
	BillingDunningGrace = 72 * time.Hour

	// FreshnessTTL is how long a verified license is considered fresh before background verification
	// rechecks it with the provider.
	FreshnessTTL = time.Hour

	keyStorePrefix   = "license/"
	hintVisibleRunes = 4
)

var (
	// ErrNotFound is a license id that no installed plugin declares, or a key that no provider
	// recognizes.
	ErrNotFound = errors.New("license not found")
	// ErrInvalidKey is an empty or malformed license key.
	ErrInvalidKey = errors.New("invalid license key")
	// ErrActivationLimitReached is a key that has used all of its machine activation slots.
	ErrActivationLimitReached = errors.New("license activation limit reached")
	// ErrExpired is a key whose subscription or validity period has expired.
	ErrExpired = errors.New("license key expired")
	// ErrDisabled is a key that the provider has disabled or revoked.
	ErrDisabled = errors.New("license key is not active")
	// ErrProductMismatch is a key that belongs to another product.
	ErrProductMismatch = errors.New("license key belongs to another product")
	// ErrProviderUnavailable is a provider that could not be reached or answered with a transient
	// failure.
	ErrProviderUnavailable = errors.New("license provider unavailable")
	// ErrKeyStore is a key store that failed or is absent on this machine.
	ErrKeyStore = errors.New("the key store failed")
)

// Contributor is one plugin that declares a license and what it unlocks when the license is active.
type Contributor struct {
	PluginID    string
	PluginName  string
	Description string
}

// ProviderOption is one service through which a person can buy a key for a license.
type ProviderOption struct {
	Provider       string
	ProviderName   string
	ProductID      string
	CheckoutURL    string
	PaymentMethods []string
}

// Declaration is a license merged across all plugins that declare its ID.
type Declaration struct {
	ID               string
	Title            string
	Contributors     []Contributor
	Providers        []manifest.LicenseProvider
	OfflineGraceDays int
}

// IsCustom reports whether the license declares no external providers and is validated by the
// plugin itself.
func (d Declaration) IsCustom() bool {
	return len(d.Providers) == 0
}

// Info is one license together with its contributors, checkout providers and activation state on
// this machine.
type Info struct {
	ID             string
	Title          string
	Contributors   []Contributor
	Providers      []ProviderOption
	Status         Status
	IsActive       bool
	HasKey         bool
	KeyHint        string
	ActiveProvider string
	InstanceName   string
	ExpiresAt      string
	VerifiedAt     string
}

// Envelope is the record stored in the key store under license/<id>.
type Envelope struct {
	Key                string
	KeyHint            string
	Provider           string
	InstanceID         string
	InstanceName       string
	MachineFingerprint string
	Status             Status
	ExpiresAt          string
	VerifiedAt         string
	LastSeenAt         string
}

const (
	envelopeFieldKey                = "key"
	envelopeFieldKeyHint            = "keyHint"
	envelopeFieldProvider           = "provider"
	envelopeFieldInstanceID         = "instanceId"
	envelopeFieldInstanceName       = "instanceName"
	envelopeFieldMachineFingerprint = "machineFingerprint"
	envelopeFieldStatus             = "status"
	envelopeFieldExpiresAt          = "expiresAt"
	envelopeFieldVerifiedAt         = "verifiedAt"
	envelopeFieldLastSeenAt         = "lastSeenAt"
)

// Evaluation is the result of evaluating a stored Envelope on this machine at an instant in time.
type Evaluation struct {
	Status          Status
	IsActive        bool
	HasKey          bool
	NeedsValidation bool
	IsClockTampered bool
	UpdatedEnvelope Envelope
	EnvelopeChanged bool
}

// GetEntryName prefixes a license ID with "license/" for the key store.
func GetEntryName(licenseID string) string {
	return keyStorePrefix + licenseID
}

// NormalizeKey trims surrounding whitespace and invisible format characters from a license key.
func NormalizeKey(key string) string {
	return strings.TrimFunc(key, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r)
	})
}

// GetKeyHint returns a masked representation of a license key showing at most the last four
// characters.
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

// EncodeEnvelope serializes an Envelope as JSON for the key store.
func EncodeEnvelope(env Envelope) (string, error) {
	fields := map[string]string{
		envelopeFieldKey:                env.Key,
		envelopeFieldKeyHint:            env.KeyHint,
		envelopeFieldProvider:           env.Provider,
		envelopeFieldMachineFingerprint: env.MachineFingerprint,
		envelopeFieldStatus:             string(env.Status),
	}
	setIfNotEmpty(fields, envelopeFieldInstanceID, env.InstanceID)
	setIfNotEmpty(fields, envelopeFieldInstanceName, env.InstanceName)
	setIfNotEmpty(fields, envelopeFieldExpiresAt, env.ExpiresAt)
	setIfNotEmpty(fields, envelopeFieldVerifiedAt, env.VerifiedAt)
	setIfNotEmpty(fields, envelopeFieldLastSeenAt, env.LastSeenAt)
	raw, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func setIfNotEmpty(fields map[string]string, key, val string) {
	if val != "" {
		fields[key] = val
	}
}

// DecodeEnvelope parses a JSON Envelope from the key store.
func DecodeEnvelope(raw string) (Envelope, error) {
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Key:                fields[envelopeFieldKey],
		KeyHint:            fields[envelopeFieldKeyHint],
		Provider:           fields[envelopeFieldProvider],
		InstanceID:         fields[envelopeFieldInstanceID],
		InstanceName:       fields[envelopeFieldInstanceName],
		MachineFingerprint: fields[envelopeFieldMachineFingerprint],
		Status:             Status(fields[envelopeFieldStatus]),
		ExpiresAt:          fields[envelopeFieldExpiresAt],
		VerifiedAt:         fields[envelopeFieldVerifiedAt],
		LastSeenAt:         fields[envelopeFieldLastSeenAt],
	}, nil
}

// FormatTime formats an instant in UTC RFC 3339 format.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ParseTime parses an RFC 3339 timestamp in UTC, returning the zero Time on empty or invalid text.
func ParseTime(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t.UTC(), true
}
