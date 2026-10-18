//go:build (darwin || freebsd || linux || netbsd || windows) && !android

package onnxruntime

import (
	"fmt"

	"github.com/ebitengine/purego"
)

// apiVersion is the OrtApi revision the plugins ask the library for.
const apiVersion = 23

// apiBase is the head of the table OrtGetApiBase returns.
type apiBase struct {
	getAPI uintptr
}

// checkAPI returns an error when the library whose OrtGetApiBase stands at
// getAPIBase serves no apiVersion table.
func checkAPI(getAPIBase uintptr) error {
	var getBase func() *apiBase
	purego.RegisterFunc(&getBase, getAPIBase)
	base := getBase()
	if base == nil || base.getAPI == 0 {
		return fmt.Errorf("OrtGetApiBase returned no table")
	}
	var getAPI func(version uint32) uintptr
	purego.RegisterFunc(&getAPI, base.getAPI)
	if getAPI(apiVersion) == 0 {
		return fmt.Errorf("the library serves no API %d, which a release older than %s lacks", apiVersion, version)
	}
	return nil
}
