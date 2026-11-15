package updates

import "testing"

func TestTheEndpointIsTheFeedOfTheChannelOrTheSettingOfTheCustomChannel(t *testing.T) {
	cases := []struct {
		name    string
		values  map[string]string
		version string
		want    string
	}{
		{"no settings on a stable build", map[string]string{}, "2026.11.5", StableEndpoint},
		{"no settings on a nightly build", map[string]string{}, "2026.11.5-nightly.2113", NightlyEndpoint},
		{"no settings on a development build", map[string]string{}, "0.0.0-dev", StableEndpoint},
		{"a blank channel on a nightly build", map[string]string{"channel": " "}, "2026.11.5-nightly.2113", NightlyEndpoint},
		{"the stable channel on a nightly build", map[string]string{"channel": "stable"}, "2026.11.5-nightly.2113", StableEndpoint},
		{"the nightly channel on a stable build", map[string]string{"channel": "nightly"}, "2026.11.5", NightlyEndpoint},
		{"the custom channel", map[string]string{"endpoint": " http://127.0.0.1:1/latest.json ", "channel": "custom"}, "2026.11.5-nightly.2113", "http://127.0.0.1:1/latest.json"},
		{"the custom channel without an address", map[string]string{"endpoint": "  ", "channel": "custom"}, "2026.11.5", ""},
		{"an endpoint on the nightly channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json", "channel": "nightly"}, "2026.11.5", NightlyEndpoint},
		{"an endpoint on a stable build with no channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json"}, "2026.11.5", StableEndpoint},
	}
	for _, c := range cases {
		if got := GetEndpoint(c.values, c.version); got != c.want {
			t.Errorf("%s: endpoint = %q, want %q", c.name, got, c.want)
		}
	}
}
