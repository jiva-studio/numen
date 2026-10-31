// Package main tests the plugin process lifecycle.
package main

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

func TestLifecycleConfigure(t *testing.T) {
	plugintest.CheckConfigure(t, "zoom", "1.5")
}
