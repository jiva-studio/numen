package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"

// textVersions is what the application does with the text versions of the files of the vaults: recording
// them, reading them, and reporting that they changed. The three work the same stores.
type textVersions struct {
	Versions *textversion.Versions
	Texts    *textversion.Texts
	Changes  *textversion.Changes
}
