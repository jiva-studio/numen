package pluginsdk

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getSDKRefusal(mod, dep string) string {
	switch {
	case !load.IsSDK(dep), dep == mod:
		return ""
	case mod == sdkModule:
		return "the base module requires no feature module"
	case dep != sdkModule:
		return "a feature module requires no other feature module"
	}
	return ""
}
