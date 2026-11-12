package plugin

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// HealthSeverity is how bad a health item is. HealthOK clears the key it is reported under, and
// HealthUnspecified is no severity.
type HealthSeverity int

const (
	HealthUnspecified HealthSeverity = iota
	HealthOK
	HealthWarning
	HealthError
)

// The keys of the health items the core reports for every plugin. A plugin cannot report under them.
const (
	HealthKeyConfigure = "configure"
	HealthKeyStart     = "start"
	HealthKeySearch    = "search"
)

// The limits of the health items of a plugin.
const (
	MaxHealthKeys      = 8
	MaxHealthKeyLength = 32
	MaxHealthText      = 120
	MaxHealthDetail    = 500
)

// AppCommandOpenSettings is the command that opens the settings, at the plugin and the setting its
// values name.
const AppCommandOpenSettings = "app:settings.open"

// The arguments of AppCommandOpenSettings.
const (
	OpenSettingsPlugin  = "plugin"
	OpenSettingsSetting = "setting"
)

// ErrInvalidHealth is a health item the core refuses.
var ErrInvalidHealth = errors.New("invalid health item")

var healthKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// HealthAction is the command a person runs to fix what a health item reports. An empty Title reads
// the title of the command.
type HealthAction struct {
	Command string
	Values  map[string]string
	Title   string
}

// HealthItem is one concern of one plugin that does not work, under the plugin's key for it.
type HealthItem struct {
	PluginID string
	Key      string
	Severity HealthSeverity
	Text     string
	Detail   string
	Action   *HealthAction
}

// IsCoreHealthKey reports whether the key is one the core reports under.
func IsCoreHealthKey(key string) bool {
	return key == HealthKeyConfigure || key == HealthKeyStart || key == HealthKeySearch
}

// CheckHealthKey returns ErrInvalidHealth for a key that is not lowercase letters, digits and hyphens
// starting with a letter, or is longer than MaxHealthKeyLength.
func CheckHealthKey(key string) error {
	if len(key) > MaxHealthKeyLength {
		return fmt.Errorf("%w: the key %q is longer than %d characters", ErrInvalidHealth, key, MaxHealthKeyLength)
	}
	if !healthKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: the key %q is not lowercase letters, digits and hyphens starting with a letter", ErrInvalidHealth, key)
	}
	return nil
}

// CheckHealthItem returns ErrInvalidHealth for an item whose key, text, detail or severity is out of
// the limits. It does not check the action.
func CheckHealthItem(item HealthItem) error {
	if err := CheckHealthKey(item.Key); err != nil {
		return err
	}
	if item.Severity != HealthWarning && item.Severity != HealthError {
		return fmt.Errorf("%w: the item %q has no severity", ErrInvalidHealth, item.Key)
	}
	if item.Text == "" {
		return fmt.Errorf("%w: the item %q has no text", ErrInvalidHealth, item.Key)
	}
	if utf8.RuneCountInString(item.Text) > MaxHealthText {
		return fmt.Errorf("%w: the text of the item %q is longer than %d characters", ErrInvalidHealth, item.Key, MaxHealthText)
	}
	if utf8.RuneCountInString(item.Detail) > MaxHealthDetail {
		return fmt.Errorf("%w: the detail of the item %q is longer than %d characters", ErrInvalidHealth, item.Key, MaxHealthDetail)
	}
	return nil
}

// FitHealthText returns the text cut to MaxHealthText characters and the whole text as the detail
// where it is longer, for an item the core writes from an error.
func FitHealthText(text string) (fitted, detail string) {
	if utf8.RuneCountInString(text) <= MaxHealthText {
		return text, ""
	}
	runes := []rune(text)
	whole := text
	if len(runes) > MaxHealthDetail {
		whole = string(runes[:MaxHealthDetail])
	}
	return string(runes[:MaxHealthText-1]) + "…", whole
}
