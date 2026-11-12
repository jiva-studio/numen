package theme

import themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"

// ListThemes lists every theme the installation offers. Every name in the list is a theme that can be applied.
type ListThemes struct {
	catalog catalog
}

func NewListThemes(sources Sources) *ListThemes {
	return &ListThemes{catalog: catalog{sources: sources}}
}

// Execute returns the shipped themes, then the themes of the enabled plugins, then the user's themes.
func (u *ListThemes) Execute() []themedomain.Theme {
	return u.catalog.list()
}
