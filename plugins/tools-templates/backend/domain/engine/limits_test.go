package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
)

func renderIn(ctx context.Context, text string) (engine.Result, error) {
	return engine.New(nil).Render(ctx, "T.md", text, getData())
}

func TestStopATemplateThatWritesTooMuch(t *testing.T) {
	for _, c := range []struct{ name, in string }{
		{"a huge range", `{{ range 1000000000 }}x{{ end }}`},
		{"a doubling text", `{{ $s := "xxxxxxxx" }}{{ range 40 }}{{ $s = replace "x" "xx" $s }}{{ end }}{{ $s }}`},
		{"a joined list", `{{ $s := "xxxxxxxx" }}{{ range 40 }}{{ $s = join $s (list $s $s) }}{{ end }}{{ $s }}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := renderIn(t.Context(), c.in)

			if !errors.Is(err, engine.ErrOutputTooLarge) || !errors.Is(err, engine.ErrTemplate) {
				t.Errorf("error = %v", err)
			}
		})
	}
	t.Run("lets a text of the size of the limit through", func(t *testing.T) {
		got, err := renderIn(t.Context(), `{{ range 1024 }}{{ range 1024 }}x{{ end }}{{ end }}`)

		if err != nil || len(got.Text) != engine.MaxOutput {
			t.Errorf("length = %d, error = %v", len(got.Text), err)
		}
	})
}

func TestStopATemplateThatRunsTooLong(t *testing.T) {
	for _, c := range []struct{ name, in string }{
		{"a huge range of calls", `{{ range 1000000000 }}{{ $_ := prompt "A" }}{{ end }}`},
		{"a template that calls itself", `{{ define "a" }}{{ template "a" }}{{ end }}{{ template "a" }}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()

			start := time.Now()
			_, err := renderIn(ctx, c.in)

			if !errors.Is(err, engine.ErrTemplate) {
				t.Errorf("error = %v", err)
			}
			if time.Since(start) > 2*time.Second {
				t.Errorf("it took %v to stop", time.Since(start))
			}
		})
	}
	t.Run("says it ran too long", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		_, err := renderIn(ctx, `{{ range 1000000000 }}{{ $_ := prompt "A" }}{{ end }}`)

		if !errors.Is(err, engine.ErrTooSlow) {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("reports a canceled call as canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := renderIn(ctx, `{{ range 1000000000 }}{{ $_ := prompt "A" }}{{ end }}`)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v", err)
		}
	})
}

func TestOfferNoFunctionThatCanReachOutOrGrowWithoutBound(t *testing.T) {
	for _, in := range []string{`{{ call .title }}`, `{{ html "<b>" }}`, `{{ js "x" }}`, `{{ printf "%1000000000d" 1 }}`, `{{ print 1 }}`, `{{ println 1 }}`} {
		t.Run(in, func(t *testing.T) {
			_, err := renderIn(t.Context(), in)

			if !errors.Is(err, engine.ErrTemplate) || !strings.Contains(err.Error(), "not available") {
				t.Errorf("error = %v", err)
			}
		})
	}
}
