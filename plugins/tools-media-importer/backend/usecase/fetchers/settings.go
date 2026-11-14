package fetchers

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

const defaultCopyMaxSize = 500 << 20

// values are the programs and the fetch settings the setting values give.
type values struct {
	Programs fetch.Config
	Fetch    fetch.Settings
}

func getDefaultValues() values {
	return values{
		Programs: fetch.Defaults(),
		Fetch:    fetch.Settings{Captions: nil, IsAutomatic: true, CopyMaxSize: defaultCopyMaxSize},
	}
}

// parseValues returns current with the settings present in settings applied. A setting that is
// absent, or whose value is not a size in megabytes, keeps its current value.
func parseValues(settings map[string]string, current values) values {
	if v, ok := settings["ytDlp"]; ok {
		current.Programs.Video = getTool(v)
	}
	if v, ok := settings["ffmpeg"]; ok {
		current.Programs.Sound = getTool(v)
	}
	if v, ok := settings["captions"]; ok {
		current.Fetch.Captions = getLanguages(v)
	}
	if v, ok := settings["automaticCaptions"]; ok {
		current.Fetch.IsAutomatic = v != "false"
	}
	if mb, err := strconv.ParseInt(settings["copyMaxSizeMb"], 10, 64); err == nil && mb >= 0 {
		current.Fetch.CopyMaxSize = mb << 20
	}
	return current
}

func getTool(path string) fetch.Tool {
	if path == "" {
		// gomutants:disable-next-line RETURN_ZERO reason="explicit nil fields equal zero Tool"
		return fetch.Tool{Command: nil, Arguments: nil, Environment: nil}
	}
	return fetch.Tool{Command: []string{path}, Arguments: nil, Environment: nil}
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
