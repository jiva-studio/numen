package load

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// SdkModule is the base module of the plugin SDK. Each feature module is a
// folder under it with a go.mod of its own, named after that folder.
const SdkModule = NumenRoot + "libs/plugin/sdk-go"

func IsSDK(path string) bool {
	return path == SdkModule || strings.HasPrefix(path, SdkModule+"/")
}
