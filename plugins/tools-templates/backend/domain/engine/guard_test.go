package engine_test

import (
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
)

func TestStopATemplateThatSpinsWithoutWriting(t *testing.T) {
	for _, c := range []struct{ name, in string }{
		{"a huge range with an empty body", `{{ range 9000000000000 }}{{ end }}`},
		{"a nested range over a split", `{{ range 1000000 }}{{ range split "" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" }}{{ end }}{{ end }}`},
		{"a template that fans out", `{{ define "a" }}{{ template "b" }}{{ template "b" }}{{ end }}{{ define "b" }}{{ template "c" }}{{ template "c" }}{{ end }}{{ define "c" }}{{ template "d" }}{{ template "d" }}{{ end }}{{ define "d" }}{{ template "e" }}{{ template "e" }}{{ end }}{{ define "e" }}{{ range 100000000 }}{{ end }}{{ end }}{{ template "a" }}`},
		{"a template that calls itself", `{{ define "a" }}{{ template "a" }}{{ end }}{{ template "a" }}`},
		{"a loop in the else branch", `{{ range list }}{{ else }}{{ range 9000000000000 }}{{ end }}{{ end }}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := runtime.NumGoroutine()

			_, err := renderIn(t.Context(), c.in)

			if !errors.Is(err, engine.ErrTooSlow) {
				t.Fatalf("error = %v", err)
			}
			for range 100 {
				if runtime.NumGoroutine() <= before {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if after := runtime.NumGoroutine(); after > before {
				t.Errorf("goroutines = %d, before = %d", after, before)
			}
		})
	}
}
