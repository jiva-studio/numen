//go:build linux

package app

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

const dmabufEnv = "WEBKIT_DISABLE_DMABUF_RENDERER"

func applyWebviewEnvironment() {
	path, err := settings.Path()
	if err != nil {
		applyDmabufEnvironment(settings.DefaultPerformance().DisableDmabufRenderer)
		return
	}
	applyDmabufEnvironment(settings.ReadDisableDmabufRenderer(path))
}

func applyDmabufEnvironment(isDisabled bool) {
	if os.Getenv(dmabufEnv) != "" || !isDisabled {
		return
	}
	if err := os.Setenv(dmabufEnv, "1"); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
	}
}
