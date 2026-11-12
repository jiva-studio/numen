// Package settings reads the values of the plugin's settings into the programs and the fetch
// settings the importer uses.
package settings

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

const defaultCopyMaxSize = 500 << 20

// Values are the programs and the fetch settings the setting values give.
type Values struct {
	Programs download.Config
	Fetch    link.Settings
}

// Defaults returns the values before any setting is read.
func Defaults() Values {
	return Values{
		Programs: download.Defaults(),
		Fetch:    link.Settings{Captions: nil, IsAutomatic: true, CopyMaxSize: defaultCopyMaxSize},
	}
}

// Parse returns current with the settings present in values applied. A setting that is absent, or
// whose value is not a size in megabytes, keeps its current value.
func Parse(values map[string]string, current Values) Values {
	if v, ok := values["ytDlp"]; ok {
		current.Programs.Video = getTool(v)
	}
	if v, ok := values["ffmpeg"]; ok {
		current.Programs.Sound = getTool(v)
	}
	if v, ok := values["captions"]; ok {
		current.Fetch.Captions = getLanguages(v)
	}
	if v, ok := values["automaticCaptions"]; ok {
		current.Fetch.IsAutomatic = v != "false"
	}
	if mb, err := strconv.ParseInt(values["copyMaxSizeMb"], 10, 64); err == nil && mb >= 0 {
		current.Fetch.CopyMaxSize = mb << 20
	}
	return current
}

func getTool(path string) download.Tool {
	if path == "" {
		// gomutants:disable-next-line RETURN_ZERO reason="explicit nil fields equal zero Tool"
		return download.Tool{Command: nil, Arguments: nil, Environment: nil}
	}
	return download.Tool{Command: []string{path}, Arguments: nil, Environment: nil}
}

func getLanguages(value string) []string {
	var out []string
	for one := range strings.SplitSeq(value, ",") {
		if one = strings.TrimSpace(one); one != "" {
			out = append(out, one)
		}
	}
	return out
}
