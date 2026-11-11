// Package settings reads the settings the host supplies to the plugin.
package settings

import (
	"fmt"
	"os"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// settingDayStart is the setting that defines when a day of review begins.
const settingDayStart = "dayStartTime"

// GetDayStart is the time of day at which a day of review begins. A day start that is not a valid
// hour is reported on the error output and gives the default.
func GetDayStart(values map[string]string) time.Duration {
	starts, err := review.ParseDayStart(values[settingDayStart])
	if err != nil {
		fmt.Fprintln(os.Stderr, "flashcards-studio:", settingDayStart, err)
	}
	return starts
}
