package manifest

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// The color schemes a theme declares. An empty appearance is a theme that follows the mode.
const (
	AppearanceLight = "light"
	AppearanceDark  = "dark"
)

// ThemeExtension is the extension of a theme stylesheet.
const ThemeExtension = ".css"

// MaxThemeSize is the largest theme stylesheet, in bytes. The text is spliced into the page before
// the window draws.
const MaxThemeSize = 256 << 10

// Theme is a stylesheet a plugin adds to the list of themes.
type Theme struct {
	// ID is the slug the theme is named by inside the plugin.
	ID string
	// Name is the title the theme is listed with.
	Name string
	// Appearance is AppearanceLight or AppearanceDark when the theme sets one color scheme, and empty
	// when it follows the mode.
	Appearance string
	// File is the slash-separated path of the stylesheet inside the plugin folder.
	File string
}

// listThemeIssues returns each rule a theme declaration breaks: a slug id listed once, a name, an
// appearance of light, dark or none, and a stylesheet path that stays inside the plugin folder.
func (c Contributes) listThemeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range c.Themes {
		switch {
		case !categoryNamePattern.MatchString(declared.ID):
			issues = append(issues, fmt.Errorf("%w: theme id %q is not lowercase words joined by hyphens", ErrInvalidContribution, declared.ID))
		case observed[declared.ID]:
			issues = append(issues, fmt.Errorf("%w: theme %q is declared twice", ErrInvalidContribution, declared.ID))
		}
		observed[declared.ID] = true
		if strings.TrimSpace(declared.Name) == "" {
			issues = append(issues, fmt.Errorf("%w: theme %q has no name", ErrInvalidContribution, declared.ID))
		}
		switch declared.Appearance {
		case "", AppearanceLight, AppearanceDark:
		default:
			issues = append(issues, fmt.Errorf("%w: theme %q has appearance %q, expected light, dark or none", ErrInvalidContribution, declared.ID, declared.Appearance))
		}
		if err := CheckThemeFile(declared.File); err != nil {
			issues = append(issues, fmt.Errorf("%w: theme %q: %w", ErrInvalidContribution, declared.ID, err))
		}
	}
	return issues
}

// CheckThemeFile reports why a path is not a stylesheet inside a plugin folder: it is empty,
// absolute, backslashed, climbs out of the folder, or does not end in ThemeExtension.
func CheckThemeFile(file string) error {
	switch {
	case file == "":
		return errors.New("file is empty")
	case strings.ContainsAny(file, `\`) || strings.ContainsRune(file, 0):
		return fmt.Errorf("file %q is not a slash-separated path", file)
	case path.IsAbs(file) || file != path.Clean(file):
		return fmt.Errorf("file %q is not a clean relative path", file)
	case file == ".." || strings.HasPrefix(file, "../"):
		return fmt.Errorf("file %q leaves the plugin folder", file)
	case !strings.HasSuffix(file, ThemeExtension) || path.Base(file) == ThemeExtension:
		return fmt.Errorf("file %q is not a %s stylesheet", file, ThemeExtension)
	}
	return nil
}
