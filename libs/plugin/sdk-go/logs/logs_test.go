package logs_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/logs"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/logs/logstest"
)

func TestClientWritesEntriesToTheHostAcrossAllLevels(t *testing.T) {
	if logs.Debug != 1 || logs.Info != 2 || logs.Warning != 3 || logs.Error != 4 {
		t.Fatalf("levels = %d, %d, %d, %d", logs.Debug, logs.Info, logs.Warning, logs.Error)
	}
	t.Setenv(hostcall.HostPortEnv, "")
	if _, err := logs.NewClientFromEnv(); !errors.Is(err, hostcall.ErrNoHost) {
		t.Fatalf("NewClientFromEnv without host = %v, want ErrNoHost", err)
	}

	host := logstest.NewHost(t)
	for name, value := range host.BuildVars("token-of-the-plugin") {
		t.Setenv(name, value)
	}
	client, err := logs.NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	cases := []logs.Entry{
		{Level: logs.Debug, Message: "debug step", Properties: map[string]any{"step": "init", "rawJson": `{"ok":true}`}},
		{Level: logs.Info, Message: "info step"},
		{Level: logs.Warning, Message: "warning step"},
		{Level: logs.Error, Message: "error step"},
	}
	for _, entry := range cases {
		if err := client.Write(t.Context(), entry); err != nil {
			t.Fatalf("Write(%+v): %v", entry, err)
		}
	}

	got := host.List()
	if len(got) != len(cases) {
		t.Fatalf("len(List()) = %d, want %d", len(got), len(cases))
	}
	for i, want := range cases {
		if got[i].Level != want.Level || got[i].Message != want.Message {
			t.Errorf("got[%d] = %+v, want %+v", i, got[i], want)
		}
	}
	if got[0].Properties["step"] != "init" {
		t.Errorf("got[0].Properties[step] = %v, want init", got[0].Properties["step"])
	}
}
