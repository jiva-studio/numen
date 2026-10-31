// Package main tests the plugin process.
package main

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

func TestLifecycleConfigure(t *testing.T) {
	plugintest.CheckConfigure(t, "theme", "sepia")
}
