package license

import (
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginEntry is the minimal plugin metadata needed to merge license declarations across manifests.
type PluginEntry struct {
	ID       string
	Name     string
	Licenses []manifest.License
}

// MergeDeclarations merges the license declarations of the plugins in plugin ID order. Each unique
// license ID produces one Declaration with all contributing plugins, merged providers and the
// shortest declared offlineGraceDays.
func MergeDeclarations(plugins []PluginEntry) []Declaration {
	ordered := slices.Clone(plugins)
	slices.SortFunc(ordered, func(a, b PluginEntry) int {
		return strings.Compare(a.ID, b.ID)
	})
	byID := make(map[string]*Declaration)
	var ids []string
	for _, plugin := range ordered {
		for _, lic := range plugin.Licenses {
			ids = mergePluginLicense(byID, ids, plugin, lic)
		}
	}
	result := make([]Declaration, 0, len(ids))
	for _, id := range ids {
		decl := *byID[id]
		if decl.Title == "" {
			decl.Title = decl.ID
		}
		result = append(result, decl)
	}
	return result
}

func mergePluginLicense(
	byID map[string]*Declaration,
	ids []string,
	plugin PluginEntry,
	lic manifest.License,
) []string {
	decl, exists := byID[lic.ID]
	if !exists {
		decl = &Declaration{
			ID:               lic.ID,
			Title:            lic.Title,
			OfflineGraceDays: getLicenseOfflineGraceDays(lic),
		}
		byID[lic.ID] = decl
		ids = append(ids, lic.ID)
	} else {
		applySecondaryLicenseFields(decl, lic)
	}
	decl.Contributors = append(decl.Contributors, Contributor{
		PluginID:    plugin.ID,
		PluginName:  plugin.Name,
		Description: lic.Description,
	})
	decl.Providers = mergeProviders(decl.Providers, lic.Providers)
	return ids
}

func applySecondaryLicenseFields(decl *Declaration, lic manifest.License) {
	if decl.Title == "" && lic.Title != "" {
		decl.Title = lic.Title
	}
	if days := getLicenseOfflineGraceDays(lic); days < decl.OfflineGraceDays {
		decl.OfflineGraceDays = days
	}
}

func mergeProviders(existing, added []manifest.LicenseProvider) []manifest.LicenseProvider {
	out := slices.Clone(existing)
	for _, prov := range added {
		idx := slices.IndexFunc(out, func(p manifest.LicenseProvider) bool {
			return p.Provider == prov.Provider
		})
		if idx < 0 {
			out = append(out, prov)
			continue
		}
		if out[idx].ProductID == "" && prov.ProductID != "" {
			out[idx].ProductID = prov.ProductID
		}
		if out[idx].URL == "" && prov.URL != "" {
			out[idx].URL = prov.URL
		}
	}
	return out
}

// EvaluateEnvelope checks a stored Envelope against the current machine fingerprint, UTC clock and
// offline grace period.
func EvaluateEnvelope(
	env Envelope,
	decl Declaration,
	currentFingerprint string,
	now time.Time,
) Evaluation {
	nowUTC := now.UTC()
	if !hasMatchingMachineKey(env, currentFingerprint) {
		return Evaluation{Status: StatusInactive, IsActive: false, HasKey: false, UpdatedEnvelope: env}
	}
	if decl.IsCustom() || env.Provider == ProviderCustom {
		updated, changed := advanceLastSeen(env, nowUTC)
		return Evaluation{
			Status:          StatusActive,
			IsActive:        true,
			HasKey:          true,
			UpdatedEnvelope: updated,
			EnvelopeChanged: changed,
		}
	}
	switch env.Status {
	case StatusExpired:
		return Evaluation{Status: StatusExpired, IsActive: false, HasKey: true, UpdatedEnvelope: env}
	case StatusDisabled, StatusInactive, StatusUnverified:
		return Evaluation{Status: StatusUnverified, IsActive: false, HasKey: true, NeedsValidation: true, UpdatedEnvelope: env}
	case StatusActive:
	}
	return evaluateActiveProviderEnvelope(env, decl, nowUTC)
}

func hasMatchingMachineKey(env Envelope, currentFingerprint string) bool {
	return NormalizeKey(env.Key) != "" &&
		env.MachineFingerprint != "" &&
		env.MachineFingerprint == currentFingerprint
}

func evaluateActiveProviderEnvelope(env Envelope, decl Declaration, nowUTC time.Time) Evaluation {
	verifiedAt, hasVerified := ParseTime(env.VerifiedAt)
	lastSeenAt, hasLastSeen := ParseTime(env.LastSeenAt)
	if !hasVerified {
		return Evaluation{
			Status:          StatusUnverified,
			IsActive:        false,
			HasKey:          true,
			NeedsValidation: true,
			UpdatedEnvelope: env,
		}
	}
	if isClockRolledBack(nowUTC, verifiedAt, lastSeenAt, hasLastSeen) {
		return Evaluation{
			Status:          StatusUnverified,
			IsActive:        false,
			HasKey:          true,
			NeedsValidation: true,
			IsClockTampered: true,
			UpdatedEnvelope: env,
		}
	}
	updated, changed := advanceLastSeen(env, nowUTC)
	if isOfflineGraceElapsed(decl.OfflineGraceDays, verifiedAt, nowUTC) {
		return Evaluation{
			Status:          StatusUnverified,
			IsActive:        false,
			HasKey:          true,
			NeedsValidation: true,
			UpdatedEnvelope: updated,
			EnvelopeChanged: changed,
		}
	}
	isHardExpired, needsExpiryCheck := evaluateExpiration(env.ExpiresAt, nowUTC)
	if isHardExpired {
		return Evaluation{
			Status:          StatusExpired,
			IsActive:        false,
			HasKey:          true,
			NeedsValidation: true,
			UpdatedEnvelope: updated,
			EnvelopeChanged: changed,
		}
	}
	return Evaluation{
		Status:          StatusActive,
		IsActive:        true,
		HasKey:          true,
		NeedsValidation: !nowUTC.Before(verifiedAt.Add(FreshnessTTL)) || needsExpiryCheck,
		UpdatedEnvelope: updated,
		EnvelopeChanged: changed,
	}
}

func isOfflineGraceElapsed(graceDays int, verifiedAt, nowUTC time.Time) bool {
	if graceDays <= 0 {
		graceDays = manifest.DefaultOfflineGraceDays
	}
	graceDeadline := verifiedAt.Add(time.Duration(graceDays) * 24 * time.Hour)
	return !nowUTC.Before(graceDeadline)
}

func getLicenseOfflineGraceDays(lic manifest.License) int {
	if lic.OfflineGraceDays == nil || *lic.OfflineGraceDays <= 0 {
		return manifest.DefaultOfflineGraceDays
	}
	return *lic.OfflineGraceDays
}

func evaluateExpiration(expiresAtStr string, nowUTC time.Time) (bool, bool) {
	expiresAt, hasExpires := ParseTime(expiresAtStr)
	if !hasExpires || nowUTC.Before(expiresAt) {
		return false, false
	}
	return !nowUTC.Before(expiresAt.Add(BillingDunningGrace)), true
}

func isClockRolledBack(nowUTC, verifiedAt, lastSeenAt time.Time, hasLastSeen bool) bool {
	toleratedNow := nowUTC.Add(ClockRollbackTolerance)
	if toleratedNow.Before(verifiedAt) {
		return true
	}
	if hasLastSeen && toleratedNow.Before(lastSeenAt) {
		return true
	}
	return false
}

func advanceLastSeen(env Envelope, nowUTC time.Time) (Envelope, bool) {
	formattedNow := FormatTime(nowUTC)
	lastSeenAt, hasLastSeen := ParseTime(env.LastSeenAt)
	if !hasLastSeen || nowUTC.After(lastSeenAt) {
		if env.LastSeenAt != formattedNow {
			env.LastSeenAt = formattedNow
			return env, true
		}
	}
	return env, false
}
