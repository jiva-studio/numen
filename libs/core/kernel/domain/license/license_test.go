package license_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestMergeDeclarationsCombinesContributorsAndProviders(t *testing.T) {
	t.Parallel()
	grace14 := 14
	grace30 := 30
	decls := license.MergeDeclarations([]license.PluginEntry{
		{
			ID:   "md.numen.b",
			Name: "Plugin B",
			Licenses: []manifest.License{{
				ID:               "supporter",
				Title:            "Supporter",
				Description:      "Unlocks B",
				OfflineGraceDays: &grace14,
				Providers: []manifest.LicenseProvider{
					{Provider: "dodo", ProductID: "pdt_sup"},
				},
			}},
		},
		{
			ID:   "md.numen.a",
			Name: "Plugin A",
			Licenses: []manifest.License{{
				ID:               "supporter",
				Description:      "Unlocks A",
				OfflineGraceDays: &grace30,
				Providers: []manifest.LicenseProvider{
					{Provider: "creem", ProductID: "prod_sup"},
				},
			}},
		},
	})
	if len(decls) != 1 {
		t.Fatalf("len(decls) = %d, want 1", len(decls))
	}
	got := decls[0]
	if got.ID != "supporter" || got.Title != "Supporter" || got.OfflineGraceDays != 14 {
		t.Fatalf("merged declaration = %+v", got)
	}
	if len(got.Contributors) != 2 || got.Contributors[0].PluginID != "md.numen.a" || got.Contributors[1].PluginID != "md.numen.b" {
		t.Fatalf("contributors = %+v", got.Contributors)
	}
	if len(got.Providers) != 2 || got.Providers[0].Provider != "creem" || got.Providers[1].Provider != "dodo" {
		t.Fatalf("providers = %+v", got.Providers)
	}
}

func TestEvaluateEnvelopeEnforcesMachineFingerprintClockAndGrace(t *testing.T) {
	t.Parallel()
	baseTime := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	decl := license.Declaration{
		ID:               "supporter",
		Title:            "Supporter",
		Providers:        []manifest.LicenseProvider{{Provider: "creem", ProductID: "prod_sup"}},
		OfflineGraceDays: 30,
	}
	validEnv := license.Envelope{
		Key:                "CREEM-KEY-814A",
		KeyHint:            "••••-814A",
		Provider:           "creem",
		InstanceID:         "li_1",
		InstanceName:       "akd-laptop",
		MachineFingerprint: "fp-current",
		Status:             license.StatusActive,
		VerifiedAt:         license.FormatTime(baseTime),
		LastSeenAt:         license.FormatTime(baseTime),
		ExpiresAt:          license.FormatTime(baseTime.Add(15 * 24 * time.Hour)),
	}

	// 1. Fresh evaluation on the same machine.
	eval := license.EvaluateEnvelope(validEnv, decl, "fp-current", baseTime.Add(30*time.Minute))
	if !eval.IsActive || !eval.HasKey || eval.Status != license.StatusActive || eval.NeedsValidation {
		t.Fatalf("fresh evaluation = %+v", eval)
	}
	if !eval.EnvelopeChanged || eval.UpdatedEnvelope.LastSeenAt != license.FormatTime(baseTime.Add(30*time.Minute)) {
		t.Fatalf("expected LastSeenAt to advance: %+v", eval)
	}

	// 2. Machine fingerprint mismatch (cloned credentials.json).
	cloned := license.EvaluateEnvelope(validEnv, decl, "fp-other-machine", baseTime.Add(30*time.Minute))
	if cloned.IsActive || cloned.HasKey || cloned.Status != license.StatusInactive {
		t.Fatalf("cloned machine evaluation = %+v, want inactive and hasKey=false", cloned)
	}

	// 3. Clock rollback beyond 48 hours.
	rolledBack := license.EvaluateEnvelope(validEnv, decl, "fp-current", baseTime.Add(-49*time.Hour))
	if rolledBack.IsActive || !rolledBack.HasKey || rolledBack.Status != license.StatusUnverified || !rolledBack.IsClockTampered {
		t.Fatalf("clock rollback evaluation = %+v, want unverified and IsClockTampered=true", rolledBack)
	}

	// 4. Offline grace exceeded (30 days without online verification).
	noExpiryEnv := validEnv
	noExpiryEnv.ExpiresAt = ""
	pastGrace := license.EvaluateEnvelope(noExpiryEnv, decl, "fp-current", baseTime.Add(31*24*time.Hour))
	if pastGrace.IsActive || pastGrace.Status != license.StatusUnverified || !pastGrace.NeedsValidation {
		t.Fatalf("past offline grace evaluation = %+v, want unverified", pastGrace)
	}

	// 5. Within 72h billing dunning window after ExpiresAt.
	inDunning := license.EvaluateEnvelope(validEnv, decl, "fp-current", baseTime.Add(16*24*time.Hour))
	if !inDunning.IsActive || inDunning.Status != license.StatusActive || !inDunning.NeedsValidation {
		t.Fatalf("dunning window evaluation = %+v, want active with NeedsValidation=true", inDunning)
	}

	// 6. Past 72h billing dunning window after ExpiresAt.
	pastDunning := license.EvaluateEnvelope(validEnv, decl, "fp-current", baseTime.Add(19*24*time.Hour))
	if pastDunning.IsActive || pastDunning.Status != license.StatusExpired {
		t.Fatalf("past dunning window evaluation = %+v, want expired", pastDunning)
	}
}

func TestEnvelopeEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()
	env := license.Envelope{
		Key:                "CREEM-1234-814A",
		KeyHint:            license.GetKeyHint("CREEM-1234-814A"),
		Provider:           "creem",
		InstanceID:         "li_42",
		InstanceName:       "workstation",
		MachineFingerprint: "fp-1",
		Status:             license.StatusActive,
	}
	encoded, err := license.EncodeEnvelope(env)
	if err != nil {
		t.Fatalf("EncodeEnvelope() error = %v", err)
	}
	decoded, err := license.DecodeEnvelope(encoded)
	if err != nil {
		t.Fatalf("DecodeEnvelope() error = %v", err)
	}
	if decoded != env {
		t.Fatalf("DecodeEnvelope() = %+v, want %+v", decoded, env)
	}
}
