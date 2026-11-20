package license

import "time"

// ActivationResult is the machine instance and state returned when a provider activates a license
// key.
type ActivationResult struct {
	Provider   string
	InstanceID string
	Status     Status
	ExpiresAt  *time.Time
}

// ValidationResult is the license state returned when validating an active machine instance.
type ValidationResult struct {
	Status    Status
	ExpiresAt *time.Time
}
