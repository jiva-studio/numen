// Package catalog lists the plugins an application has and the plugins it could have.
//
// A Source supplies entries, the Service merges the entries of every source by plugin id, and
// the ports here are declared for the adapters that satisfy them.
package catalog

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// State is the status of a plugin in the application.
type State string

// The states of an entry.
const (
	// StateInstalled is a plugin on disk that is enabled.
	StateInstalled State = "installed"
	// StateDisabled is a plugin on disk that is switched off.
	StateDisabled State = "disabled"
	// StateAvailable is a plugin that is not on disk and that a source offers.
	StateAvailable State = "available"
)

// IsOnDisk reports whether the plugin is on disk, enabled or not.
func (s State) IsOnDisk() bool {
	return s == StateInstalled || s == StateDisabled
}

// ErrNotFound is a plugin that no source lists, or a kind of tab that no plugin provides.
var ErrNotFound = errors.New("not found in the catalog")

// ErrAlreadyOnDisk is a plugin that is on disk and is asked to be installed.
var ErrAlreadyOnDisk = errors.New("plugin is already on disk")

// Entry is one plugin as the catalog knows it.
type Entry struct {
	ID          string
	Name        string
	Version     string
	Description string
	Category    string
	Contributes manifest.Contributes
	// Source names the source the entry comes from.
	Source string
	State  State
	// Requirements are what the plugin requires of the application.
	Requirements manifest.Requirements
	// Incompatibility is why the application cannot run the plugin. Nil is a plugin it can run. The
	// Service sets it and a source leaves it nil.
	Incompatibility *manifest.IncompatibleError
}

// NewEntry returns the entry of a plugin whose manifest a source has read.
func NewEntry(source string, state State, m manifest.Manifest) Entry {
	return Entry{
		ID:              m.ID,
		Name:            m.Name,
		Version:         m.Version,
		Description:     m.Description,
		Category:        m.Contributes.Category,
		Contributes:     m.Contributes,
		Source:          source,
		State:           state,
		Requirements:    m.GetRequirements(),
		Incompatibility: nil,
	}
}

// SourceError is a source that could not be read in full.
type SourceError struct {
	Source string
	Err    error
}

// Error names the source and the cause.
func (e *SourceError) Error() string {
	return fmt.Sprintf("catalog source %s: %v", e.Source, e.Err)
}

// Unwrap returns the cause.
func (e *SourceError) Unwrap() error {
	return e.Err
}
