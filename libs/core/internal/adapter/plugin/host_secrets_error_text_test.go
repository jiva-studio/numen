package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode"

	"connectrpc.com/connect"

	pluginport "github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// pluginWords is what every plugin of these tests says besides the values.
const pluginWords = "rejected the key"

// syncBuffer is a buffer the logger may write to from any goroutine.
type syncBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// captureLogs returns what the standard logger and slog write while the test runs.
func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	logs := &syncBuffer{}
	previousSlog, previousFlags, previousWriter := slog.Default(), log.Flags(), log.Writer()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	log.SetOutput(logs)
	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
		log.SetFlags(previousFlags)
		log.SetOutput(previousWriter)
	})
	return logs
}

// assertOnlyPluginAndStatus checks that text holds nothing the plugin said: no value, no quoted form
// of a value, no capital letter of a value (a piece of it) and none of the plugin's words.
func assertOnlyPluginAndStatus(t *testing.T, where, text string, values []string) {
	t.Helper()
	for _, value := range values {
		if value != "" && strings.Contains(text, value) {
			t.Errorf("%s holds the value %q: %q", where, value, text)
		}
		if quoted := strconv.Quote(value); value != "" && strings.Contains(text, quoted) {
			t.Errorf("%s holds the quoted value %s: %q", where, quoted, text)
		}
		for _, r := range value {
			if unicode.IsUpper(r) && strings.ContainsRune(text, r) {
				t.Errorf("%s holds %q, a piece of the value %q: %q", where, r, value, text)
			}
		}
	}
	for _, phrase := range []string{pluginWords, "rejected", "[removed]"} {
		if strings.Contains(text, phrase) {
			t.Errorf("%s holds the text of the plugin (%q): %q", where, phrase, text)
		}
	}
}

// getErrorTextCases returns the values a plugin may put in its error text, and how it puts them.
func getErrorTextCases() []errorTextCase {
	return []errorTextCase{
		{"a value", []string{"ZKWQ"}, func(v []string) string { return pluginWords + " " + v[0] }},
		{"two values where one is a prefix of the other", []string{"ZKWQ", "ZKWQXVJ"},
			func(v []string) string { return pluginWords + " " + v[1] }},
		{"the shorter of two values it starts", []string{"ZKWQXVJ", "ZKWQ"},
			func(v []string) string { return pluginWords + " " + v[0] }},
		{"a value with quotes and a backslash, escaped", []string{`ZK"W\Q`},
			func(v []string) string { return fmt.Sprintf("%s %q", pluginWords, v[0]) }},
		{"a value of one character", []string{"Q"}, func(_ []string) string { return pluginWords + " Q" }},
		{"an empty value", []string{""}, func(_ []string) string { return pluginWords + " " }},
	}
}

type errorTextCase struct {
	name   string
	values []string
	say    func(values []string) string
}

// newErrorTextHost returns a host whose plugin may read one secret for each of the values.
func newErrorTextHost(t *testing.T, values []string) (*Host, *configured) {
	t.Helper()
	items := make(map[string]string, len(values))
	secrets := make([]pluginport.Secret, 0, len(values))
	for i, value := range values {
		id := fmt.Sprintf("s%d", i)
		items["secret/"+id] = value
		secrets = append(secrets, pluginport.Secret{
			ID: id, Name: fmt.Sprintf("Name %d", i), Group: "openrouter",
			Allowed: []pluginport.SecretPermission{{Plugin: secretPluginID, Name: declared}},
		})
	}
	host, probe := newSecretHost(t, newKeys(items))
	host.SetSecretSource(pluginport.SecretSource{List: func() ([]pluginport.Secret, error) { return secrets, nil }})
	return host, probe
}

// The error of a failed Configure holds the id of the plugin and the status code of the call and
// nothing the plugin said, whatever the plugin put in its text. Neither the report to the error
// handler, the failure passed on nor a log line holds it.
func TestTheErrorOfAFailedConfigureContainsNoTextOfThePlugin(t *testing.T) {
	for _, c := range getErrorTextCases() {
		t.Run(c.name, func(t *testing.T) {
			host, probe := newErrorTextHost(t, c.values)
			reported, failed, logs := &collectErrors{}, &collectFailures{}, captureLogs(t)
			host.SetErrorHandler(reported.add)
			host.SetOnPluginFailed(failed.add)
			probe.setFailure(connect.NewError(connect.CodeInternal, errors.New(c.say(c.values))))

			err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })

			if err == nil {
				t.Fatal("the operation returned nil although the plugin did not accept the list")
			}
			if !strings.Contains(err.Error(), secretPluginID) || !strings.Contains(err.Error(), connect.CodeInternal.String()) {
				t.Errorf("the error %q does not hold the plugin id and the status code", err)
			}
			assertOnlyPluginAndStatus(t, "the returned error", err.Error(), c.values)
			for _, text := range reported.texts {
				assertOnlyPluginAndStatus(t, "an error reported to the handler", text, c.values)
			}
			for _, failure := range failed.failures {
				assertOnlyPluginAndStatus(t, "a failure passed on", fmt.Sprintf("%+v", failure), c.values)
			}
			assertOnlyPluginAndStatus(t, "the log", logs.String(), c.values)
		})
	}
}

// The error the activation reports when the Configure of the activation fails holds the same.
func TestTheReportOfAFailedActivationConfigureContainsNoTextOfThePlugin(t *testing.T) {
	for _, c := range getErrorTextCases() {
		t.Run(c.name, func(t *testing.T) {
			host, probe := newErrorTextHost(t, c.values)
			reported, logs := &collectErrors{}, captureLogs(t)
			host.SetErrorHandler(reported.add)
			probe.setFailure(connect.NewError(connect.CodeInternal, errors.New(c.say(c.values))))

			if err := prepareActivation(t, host, false)(); err != nil {
				t.Fatal(err)
			}

			if reported.count() != 1 {
				t.Fatalf("%d failures reported, want 1", reported.count())
			}
			text := reported.texts[0]
			if !strings.Contains(text, secretPluginID) || !strings.Contains(text, connect.CodeInternal.String()) {
				t.Errorf("the report %q does not hold the plugin id and the status code", text)
			}
			assertOnlyPluginAndStatus(t, "the report", text, c.values)
			assertOnlyPluginAndStatus(t, "the log", logs.String(), c.values)
		})
	}
}
