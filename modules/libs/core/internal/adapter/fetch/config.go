package fetch

import (
	"maps"
	"os"
	"slices"
)

// Config is what an installation says about reaching an address.
//
// Every key here is a machine's answer: where a tool is, what it may be handed,
// and how much of what is at an address is kept on this disk. None of it is
// turned in the window.
type Config struct {
	// FetchUnasked is whether a link note the vault holds nothing fetched for
	// is fetched on its own. Off: reaching off the machine is a gesture, and a
	// note written by hand in another editor is not one.
	FetchUnasked *bool `json:"fetch_unasked"`

	// Captions are the languages published words are preferred in, best first.
	// Empty takes whatever the video calls its own.
	Captions []string `json:"captions"`

	// AutomaticCaptions is whether words a machine wrote count where a person
	// published none. On.
	AutomaticCaptions *bool `json:"automatic_captions"`

	// CopyMaxSizeMB is how large a copy may be. Above it, a copy asked for says
	// the size it was refused at and nothing is fetched.
	CopyMaxSizeMB int `json:"copy_max_size_mb"`

	// CopiesToVault is whether a copy is kept beside the note as a file of the
	// person's own. Off: a copy is fetchable again from the address the note
	// carries, so it is kept in the application's own folder, where losing it
	// costs a fetch.
	CopiesToVault *bool `json:"copies_to_vault"`

	// Video is what is run to reach a video, and Sound what brings a container
	// to what a transcriber opens. Each is a command and what it is started
	// through; empty asks the path.
	Video Tool `json:"yt_dlp"`
	Sound Tool `json:"ffmpeg"`
}

// A Tool is a program this machine holds.
//
// It is a command, so a machine that writes the path afresh at every build
// names whatever does know where the tool is. Arguments
// are handed to every run before its own: what answers for a person at a site
// that refuses an unattended fetch — the cookies of a browser, a token, a proxy
// — is that machine's and is passed through as it stands.
//
// Environment is set on every run, over what this process was started with. A
// tool that finds a machine's certificates, its cache or its proxy by an
// environment variable is told which, and a machine that keeps none of those
// where the tool looks says so here.
type Tool struct {
	Command     []string          `json:"command"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment"`
}

// env is what a run of this tool is started with: this process's own, and what
// the settings name over it. Naming none is this process's own.
func (t Tool) env() []string {
	if len(t.Environment) == 0 {
		return nil
	}
	out := os.Environ()
	for _, name := range slices.Sorted(maps.Keys(t.Environment)) {
		out = append(out, name+"="+t.Environment[name])
	}
	return out
}

// Defaults fetch nothing unasked, prefer the words a person published, and keep
// a copy in the application's own folder.
func Defaults() Config {
	automatic := true
	off := false
	return Config{
		FetchUnasked:      &off,
		AutomaticCaptions: &automatic,
		CopyMaxSizeMB:     DefaultCopyMaxSizeMB,
		CopiesToVault:     &off,
	}
}

// DefaultCopyMaxSizeMB is how large a copy may be by default. An hour of video is
// under it, and a film is not.
const DefaultCopyMaxSizeMB = 500

// Unasked is whether a link note nothing has been fetched for is fetched on its
// own.
func (c Config) Unasked() bool { return c.FetchUnasked != nil && *c.FetchUnasked }

// Automatic is whether words a machine wrote count.
func (c Config) Automatic() bool { return c.AutomaticCaptions == nil || *c.AutomaticCaptions }

// ToVault is whether a copy is kept beside the note.
func (c Config) ToVault() bool { return c.CopiesToVault != nil && *c.CopiesToVault }

// CopyBytes is how large a copy may be, in bytes. Nothing is no limit.
func (c Config) CopyBytes() int64 {
	if c.CopyMaxSizeMB <= 0 {
		return 0
	}
	return int64(c.CopyMaxSizeMB) << 20
}
