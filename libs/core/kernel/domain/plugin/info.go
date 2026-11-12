package plugin

import "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

// ReaderInfo is a reader of a plugin and the files it opens.
type ReaderInfo struct {
	ID         string
	Extensions []string
}

// ConfigPerformanceProfile is the entry of the Configure values that holds the performance profile; plugin-sdk-go names it performance.ConfigKey.
const ConfigPerformanceProfile = "host.performanceProfile"

// The limits of the search index calls.
const (
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
)

// Info describes a discovered plugin and its declarative settings.
type Info struct {
	ID            string
	Name          string
	Version       string
	Description   string
	Capabilities  []string
	Settings      []Setting
	IsActive      bool
	Permissions   []string
	Failure       *Failure
	HasFrontend   bool
	HasBackend    bool
	Readers       []ReaderInfo
	SettingsViews []manifest.SettingsView
}
