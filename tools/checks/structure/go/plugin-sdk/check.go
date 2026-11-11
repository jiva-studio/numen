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
	case mod == load.SdkModule:
		return "the base module requires no feature module"
	case dep != load.SdkModule:
		return "a feature module requires no other feature module"
	}
	return ""
}
