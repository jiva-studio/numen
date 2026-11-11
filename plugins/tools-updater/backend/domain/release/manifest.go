// Package release reads the release manifest the application publishes as latest.json and decides
// whether it names a build newer than the running one.
package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// File is one downloadable file of a release.
type File struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

// Manifest is the release manifest: the version, the build number and the files of the newest build
// of a channel.
type Manifest struct {
	Version  string `json:"version"`
	Build    Build  `json:"build"`
	Commit   string `json:"commit"`
	Stamp    string `json:"stamp"`
	Channel  string `json:"channel"`
	Notes    string `json:"notes"`
	Released string `json:"released"`
	Files    []File `json:"files"`
}

// Build is the build number of a release. The manifest writes it as a number or as text.
type Build int64

// buildStamp matches the build number at the end of a stamp such as `0.4.0-alpha.1-364-283d1d5`
// or `2026.10.1-beta.920`.
var buildStamp = regexp.MustCompile(`(?:-beta\.|\b)(\d+)(?:-[a-f0-9]{7})?$`)

// ParseBuild returns the build number a text holds: a plain number, or the number a stamp ends in.
// A text that holds none, such as a version without a stamp, is 0.
func ParseBuild(text string) Build {
	clean := strings.TrimSpace(text)
	if strings.Contains(clean, ".") && !strings.Contains(clean, "-") {
		return 0
	}
	candidate := clean
	if match := buildStamp.FindStringSubmatch(clean); match != nil {
		candidate = match[1]
	}
	number, err := strconv.ParseInt(candidate, 10, 64)
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="buildStamp extracts unsigned digits and zero returns Build(0) on both branches."
	if err != nil || number <= 0 {
		return 0
	}
	return Build(number)
}

// UnmarshalJSON reads a number or a text. A value that is neither names no build and is 0.
func (b *Build) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("read build number: %w", err)
	}
	switch decoded := value.(type) {
	case json.Number:
		*b = ParseBuild(decoded.String())
	case string:
		*b = ParseBuild(decoded)
	default:
		*b = 0
	}
	return nil
}

// Read decodes the manifest from JSON.
func Read(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("read release manifest: %w", err)
	}
	return manifest, nil
}
