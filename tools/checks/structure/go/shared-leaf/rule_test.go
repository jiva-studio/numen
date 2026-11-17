package sharedleaf

import (
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/mod/modfile"
)

func getSharedLeafViolations(dir string, file *modfile.File) []string {
	return load.GetRequirementViolations(dir, file, load.GetSharedLeafRefusal)
}
