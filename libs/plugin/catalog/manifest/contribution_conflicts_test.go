package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func plugin(id string, contributes manifest.Contributes) manifest.Discovered {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: id, Frontend: &manifest.Frontend{}, Contributes: contributes}}
}

func tabKinds(kinds ...string) []manifest.TabKind {
	var out []manifest.TabKind
	for _, kind := range kinds {
		out = append(out, manifest.TabKind{Kind: kind, Title: kind, Icon: "", When: "", Rank: domain.RankDefault, IsNoteEditor: false})
	}
	return out
}

func TestListContributionConflicts(t *testing.T) {
	cases := map[string]struct {
		plugins []manifest.Discovered
		want    []string
	}{
		"the same tab kind in two plugins": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("deck")}),
				plugin("md.example.b", manifest.Contributes{TabKinds: tabKinds("deck")}),
			},
			want: []string{`plugin "md.example.b" declares the tab kind "deck", which plugin "md.example.a" already declares`},
		},
		"the same reader id in two plugins": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{Readers: []manifest.Reader{{ID: "pdf"}}}),
				plugin("md.example.b", manifest.Contributes{Readers: []manifest.Reader{{ID: "pdf"}}}),
			},
			want: []string{`plugin "md.example.b" declares the reader "reader:pdf", which plugin "md.example.a" already declares`},
		},
		"a reader that takes the tab kind of another plugin": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("reader:pdf")}),
				plugin("md.example.b", manifest.Contributes{Readers: []manifest.Reader{{ID: "pdf"}}}),
			},
			want: []string{`plugin "md.example.b" declares the reader "reader:pdf", which plugin "md.example.a" already declares`},
		},
		"a reader and its tab kind in one plugin": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("reader:pdf"), Readers: []manifest.Reader{{ID: "pdf"}}}),
			},
		},
		"a reader with prefix that takes the tab kind of another plugin": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("reader:pdf")}),
				plugin("md.example.b", manifest.Contributes{Readers: []manifest.Reader{{ID: "reader:pdf"}}}),
			},
			want: []string{`plugin "md.example.b" declares the reader "reader:pdf", which plugin "md.example.a" already declares`},
		},
		"a refused plugin owns nothing": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("deck")}),
				plugin("md.example.b", manifest.Contributes{TabKinds: tabKinds("deck", "extra")}),
				plugin("md.example.c", manifest.Contributes{TabKinds: tabKinds("extra")}),
				plugin("md.example.d", manifest.Contributes{TabKinds: tabKinds("extra")}),
			},
			want: []string{
				`plugin "md.example.b" declares the tab kind "deck"`,
				`plugin "md.example.d" declares the tab kind "extra", which plugin "md.example.c" already declares`,
			},
		},
		"distinct declarations": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{TabKinds: tabKinds("deck")}),
				plugin("md.example.b", manifest.Contributes{TabKinds: tabKinds("cards")}),
			},
		},
		"two plugins contributing to the same license id with matching provider productId": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "A", Providers: []manifest.LicenseProvider{{Provider: "creem", ProductID: "prod_1"}},
				}}}),
				plugin("md.example.b", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "B", Providers: []manifest.LicenseProvider{{Provider: "creem", ProductID: "prod_1", URL: "https://creem.io/buy/1"}},
				}}}),
			},
		},
		"two plugins declaring different productId for the same license provider": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "A", Providers: []manifest.LicenseProvider{{Provider: "creem", ProductID: "prod_1"}},
				}}}),
				plugin("md.example.b", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "B", Providers: []manifest.LicenseProvider{{Provider: "creem", ProductID: "prod_2"}},
				}}}),
			},
			want: []string{`plugin "md.example.b" declares the license provider "sync/creem", which plugin "md.example.a" already declares`},
		},
		"two plugins declaring different url for the same license provider": {
			plugins: []manifest.Discovered{
				plugin("md.example.a", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "A", Providers: []manifest.LicenseProvider{{Provider: "dodo", URL: "https://checkout.dodopayments.com/buy/1"}},
				}}}),
				plugin("md.example.b", manifest.Contributes{Licenses: []manifest.License{{
					ID: "sync", Description: "B", Providers: []manifest.LicenseProvider{{Provider: "dodo", URL: "https://checkout.dodopayments.com/buy/2"}},
				}}}),
			},
			want: []string{`plugin "md.example.b" declares the license provider "sync/dodo", which plugin "md.example.a" already declares`},
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			conflicts := manifest.ListContributionConflicts(one.plugins, manifest.CoreClaims{})

			if len(conflicts) != len(one.want) {
				t.Fatalf("conflicts = %v, want %d", conflicts, len(one.want))
			}
			for i, conflict := range conflicts {
				if !strings.Contains(conflict.Error(), one.want[i]) {
					t.Errorf("conflict = %q, want it to contain %q", conflict, one.want[i])
				}
				if !errors.Is(conflict, manifest.ErrInvalidContribution) {
					t.Errorf("conflict %v is not an invalid contribution", conflict)
				}
			}
		})
	}
}
