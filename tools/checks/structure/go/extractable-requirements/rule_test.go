package extractablerequirements

import (
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/mod/modfile"
)

func getExtractableViolations(dir string, file *modfile.File) []string {
	mod := load.GetModulePath(file)
	return load.GetRequirementViolations(dir, file, func(dep string) string {
		return load.GetExtractableRefusal(mod, dep)
	})
}
