package engine_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
)

func TestRefuseAFieldAskedInTwoWays(t *testing.T) {
	for _, c := range []struct{ name, in string }{
		{"a prompt and a select", `{{ prompt "Kind" }}{{ select "Kind" "a" "b" }}`},
		{"a select and a prompt", `{{ select "Kind" "a" "b" }}{{ prompt "Kind" }}`},
		{"two selects with other options", `{{ select "Kind" "a" "b" }}{{ select "Kind" "a" "c" }}`},
		{"two selects with the options in another order", `{{ select "Kind" "a" "b" }}{{ select "Kind" "b" "a" }}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := engine.New(nil).ListFields(t.Context(), "T.md", c.in, getData())

			if !errors.Is(err, engine.ErrTemplate) || !strings.Contains(err.Error(), `"Kind"`) || !strings.Contains(err.Error(), "differently") {
				t.Errorf("error = %v", err)
			}
		})
	}
	t.Run("takes the same field asked again in the same way", func(t *testing.T) {
		fields, err := engine.New(nil).ListFields(t.Context(), "T.md", `{{ prompt "A" "x" }}{{ prompt "A" "y" }}{{ select "K" "a" "b" }}{{ select "K" "a" "b" }}`, getData())

		if err != nil || len(fields) != 2 {
			t.Errorf("fields = %+v, %v", fields, err)
		}
	})
}

func TestRefuseAFieldTheFirstRunDidNotMeet(t *testing.T) {
	const text = `{{ if eq (select "Kind" "call" "visit") "visit" }}{{ prompt "Address" }}{{ end }}`

	t.Run("fails where an answer leads to a field the defaults do not reach", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Kind": "visit", "Address": "Main St"}

		_, err := engine.New(nil).Render(t.Context(), "T.md", text, data)

		if !errors.Is(err, engine.ErrTemplate) || !strings.Contains(err.Error(), `"Address"`) {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("renders when the defaults reach every field", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Kind": "call"}

		if got := renderWith(t, text, data); got != "" {
			t.Errorf("text = %q", got)
		}
	})
}

func TestKeepAnIdentifierForTheWholeRequest(t *testing.T) {
	t.Run("gives the same id to the call of both passes", func(t *testing.T) {
		got := render(t, `{{ $a := uuid }}{{ prompt "X" $a }}|{{ $a }}`)

		before, after, _ := strings.Cut(got, "|")
		if before == "" || before != after {
			t.Errorf("the default %q and the id %q differ", before, after)
		}
	})
	t.Run("gives each call its own id", func(t *testing.T) {
		got := render(t, `{{ uuid }}|{{ uuid }}`)

		first, second, _ := strings.Cut(got, "|")
		if first == "" || first == second {
			t.Errorf("ids = %q and %q", first, second)
		}
	})
	t.Run("gives each render its own ids", func(t *testing.T) {
		if first, second := render(t, "{{ uuid }}"), render(t, "{{ uuid }}"); first == second {
			t.Errorf("both renders gave %q", first)
		}
	})
}
