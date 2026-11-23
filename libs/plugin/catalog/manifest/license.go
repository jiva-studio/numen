package manifest

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Offline grace window bounds for a license when the provider cannot be reached after verification.
const (
	DefaultOfflineGraceDays = 30
	MinOfflineGraceDays     = 1
	MaxOfflineGraceDays     = 365
)

// Known license provider identifiers.
const (
	LicenseProviderCreem = "creem"
	LicenseProviderDodo  = "dodo"
)

var licenseIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// LicenseProvider is one service through which a license is bought and verified. At least one of
// ProductID or URL is set.
type LicenseProvider struct {
	Provider  string
	ProductID string
	URL       string
}

// License is a license a plugin contributes to, with the features the plugin unlocks when active and
// the services that sell and verify keys for it.
type License struct {
	ID               string
	Title            string
	Description      string
	Providers        []LicenseProvider
	OfflineGraceDays *int
}

func listKnownLicenseProviders() []string {
	return []string{LicenseProviderCreem, LicenseProviderDodo}
}

func (m Manifest) listLicenseIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, one := range m.Contributes.Licenses {
		switch {
		case !licenseIDPattern.MatchString(one.ID):
			issues = append(issues, newLicenseIssue("id %q is not lowercase words joined by hyphens", one.ID))
		case observed[one.ID]:
			issues = append(issues, newLicenseIssue("%q is declared twice", one.ID))
		}
		observed[one.ID] = true
		if one.Title != strings.TrimSpace(one.Title) {
			issues = append(issues, newLicenseIssue("%q: title %q has surrounding space", one.ID, one.Title))
		}
		switch {
		case strings.TrimSpace(one.Description) == "":
			issues = append(issues, newLicenseIssue("%q has no description", one.ID))
		case one.Description != strings.TrimSpace(one.Description):
			issues = append(issues, newLicenseIssue("%q: description %q has surrounding space", one.ID, one.Description))
		}
		if one.OfflineGraceDays != nil && (*one.OfflineGraceDays < MinOfflineGraceDays || *one.OfflineGraceDays > MaxOfflineGraceDays) {
			issues = append(issues, newLicenseIssue(
				"%q: offlineGraceDays %d is not between %d and %d",
				one.ID,
				*one.OfflineGraceDays,
				MinOfflineGraceDays,
				MaxOfflineGraceDays,
			))
		}
		issues = append(issues, listLicenseProviderIssues(one)...)
	}
	return issues
}

func listLicenseProviderIssues(one License) []error {
	var issues []error
	observed := map[string]bool{}
	for _, p := range one.Providers {
		switch {
		case !slices.Contains(listKnownLicenseProviders(), p.Provider):
			issues = append(issues, newLicenseIssue("%q: provider %q is not one of %s", one.ID, p.Provider, strings.Join(listKnownLicenseProviders(), ", ")))
		case observed[p.Provider]:
			issues = append(issues, newLicenseIssue("%q: provider %q is declared twice", one.ID, p.Provider))
		}
		observed[p.Provider] = true
		if p.ProductID == "" && p.URL == "" {
			issues = append(issues, newLicenseIssue("%q: provider %q needs productId or url", one.ID, p.Provider))
			continue
		}
		if p.ProductID != "" && !isValidLicenseProductID(p.ProductID) {
			issues = append(issues, newLicenseIssue("%q: provider %q productId %q contains space or /", one.ID, p.Provider, p.ProductID))
		}
		if p.URL != "" && (strings.IndexFunc(p.URL, unicode.IsSpace) >= 0 || !domain.IsHTTPSAddress(p.URL)) {
			issues = append(issues, newLicenseIssue("%q: provider %q url %q is not an https address", one.ID, p.Provider, p.URL))
		}
	}
	return issues
}

func isValidLicenseProductID(id string) bool {
	return strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || r == '/' }) < 0
}

func newLicenseIssue(format string, args ...any) error {
	return fmt.Errorf("%w: license %s", ErrInvalidContribution, fmt.Sprintf(format, args...))
}
