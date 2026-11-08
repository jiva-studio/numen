package server

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

const redactedValue = "invented-harbor-key"

func TestASecretPrintedWithAnyVerbShowsNoValue(t *testing.T) {
	one := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: redactedValue}
	cases := map[string]string{
		"%v":                            fmt.Sprintf("%v", one),
		"%+v":                           fmt.Sprintf("%+v", one),
		"%s":                            fmt.Sprintf("%s", one),
		"%#v":                           fmt.Sprintf("%#v", one),
		"%v of a pointer":               fmt.Sprintf("%v", &one),
		"%+v of a list":                 fmt.Sprintf("%+v", []Secret{one}),
		"%#v of the secrets":            fmt.Sprintf("%#v", Secrets{SecretAPIKey: {one}}),
		"%v of a struct that holds one": fmt.Sprintf("%+v", struct{ Key Secret }{one}),
	}
	for verb, printed := range cases {
		if strings.Contains(printed, redactedValue) {
			t.Errorf("%s prints the value: %s", verb, printed)
		}
	}
}

func TestASecretLoggedWithTheStandardLoggerShowsNoValue(t *testing.T) {
	one := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: redactedValue}
	var out bytes.Buffer
	logger := log.New(&out, "", 0)

	logger.Printf("configure: %v", one)
	logger.Println(one)
	logger.Print(Secrets{SecretAPIKey: {one}})

	if strings.Contains(out.String(), redactedValue) {
		t.Errorf("the log holds the value:\n%s", out.String())
	}
}
