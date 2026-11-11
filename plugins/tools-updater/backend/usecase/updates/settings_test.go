package updates

import "testing"

func TestTheEndpointIsTheFeedOfTheChannelOrTheSettingOfTheCustomChannel(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"no settings", map[string]string{}, StableEndpoint},
		{"the stable channel", map[string]string{"channel": "stable"}, StableEndpoint},
		{"the beta channel", map[string]string{"channel": "beta"}, BetaEndpoint},
		{"the custom channel", map[string]string{"endpoint": " http://127.0.0.1:1/latest.json ", "channel": "custom"}, "http://127.0.0.1:1/latest.json"},
		{"the custom channel without an address", map[string]string{"endpoint": "  ", "channel": "custom"}, ""},
		{"an endpoint on the beta channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json", "channel": "beta"}, BetaEndpoint},
		{"an endpoint on the stable channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json"}, StableEndpoint},
	}
	for _, c := range cases {
		if got := GetEndpoint(c.values); got != c.want {
			t.Errorf("%s: endpoint = %q, want %q", c.name, got, c.want)
		}
	}
}
