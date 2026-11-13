package editor

import (
	"net/http"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestTheCallerHeaderNamesThePartOfTheWindowThatAsks(t *testing.T) {
	cases := []struct {
		tag  string
		want plugindomain.Caller
	}{
		{tag: "search", want: plugindomain.Caller{Who: "Search"}},
		{tag: "settings", want: plugindomain.Caller{Who: "Settings"}},
		{tag: "plugin:org.example.orchard", want: plugindomain.Caller{PluginID: "org.example.orchard"}},
		{tag: "plugin:", want: plugindomain.Caller{Who: plugindomain.WhoWindow}},
		{tag: "cellar", want: plugindomain.Caller{Who: plugindomain.WhoWindow}},
		{tag: "", want: plugindomain.Caller{Who: plugindomain.WhoWindow}},
	}
	for _, c := range cases {
		header := http.Header{}
		if c.tag != "" {
			header.Set(CallerHeader, c.tag)
		}
		if got := getWindowCaller(header); got != c.want {
			t.Errorf("the tag %q names %+v, want %+v", c.tag, got, c.want)
		}
	}
}
