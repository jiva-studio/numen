// Package download provides configuration types for content downloading.
package download

// Config is what an installation says about reaching an address.
type Config struct {
	// Captions are the languages published words are preferred in, best first.
	// Empty takes whatever the video calls its own.
	Captions []string `json:"captions"`

	// AutomaticCaptions is whether words a machine wrote count where a person
	// published none.
	AutomaticCaptions *bool `json:"automatic_captions"`

	// CopyMaxSizeMB is how large a copy may be. Above it, a copy asked for says
	CopyMaxSizeMB int `json:"copy_max_size_mb"`

	// CopiesToVault is whether a copy is kept beside the note as a file of the
	// person's own.
	CopiesToVault *bool `json:"copies_to_vault"`
}

// Defaults prefer the words a person published, and keep a copy in the
// application's own folder.
func Defaults() Config {
	automatic := true
	off := false
	return Config{
		AutomaticCaptions: &automatic,
		CopyMaxSizeMB:     DefaultCopyMaxSizeMB,
		CopiesToVault:     &off,
	}
}

// DefaultCopyMaxSizeMB is how large a copy may be by default.
const DefaultCopyMaxSizeMB = 500

// CanUseAutomaticCaptions is whether words a machine wrote count.
func (c Config) CanUseAutomaticCaptions() bool {
	return c.AutomaticCaptions == nil || *c.AutomaticCaptions
}

// HasCopiesInVault is whether a copy is kept beside the note.
func (c Config) HasCopiesInVault() bool { return c.CopiesToVault != nil && *c.CopiesToVault }

// CopyBytes is how large a copy may be, in bytes. Nothing is no limit.
func (c Config) CopyBytes() int64 {
	if c.CopyMaxSizeMB <= 0 {
		return 0
	}
	return int64(c.CopyMaxSizeMB) << 20
}
