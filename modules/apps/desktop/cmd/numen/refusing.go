package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// refuse draws what stopped the application, in a window.
//
// A person who opened numen from a dock has no terminal, and a message written
// to one is a window that never appeared. Whatever this build could not do, it
// says here, with what it knows about the state it found.
func refuse(cfg container.Config, why error) {
	page, err := refusal{}.page(cfg, why)
	if err != nil {
		return
	}

	app := application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(page)
			}),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen",
		Width:  720,
		Height: 520,
		URL:    "/",
	})
	_ = app.Run()
}

// refusal is what the page says: the sentence, the facts behind it, and what to
// do about it.
type refusal struct {
	Says  string
	Facts []fact
	Do    string
}

type fact struct{ Name, Value string }

// page is the refusal as one document. Everything it needs is inside it: a
// build that could not open its own index cannot serve its own interface
// either.
func (refusal) page(cfg container.Config, why error) ([]byte, error) {
	said := refusal{Says: why.Error()}

	var ahead *index.Ahead
	if errors.As(why, &ahead) {
		said.Says = "This index was written by a later version of numen."
		said.Facts = append(said.Facts,
			fact{"schema the index holds", fmt.Sprint(ahead.Held)},
			fact{"schema this build knows", fmt.Sprint(ahead.Known)},
		)
		said.Do = "Update numen to the version that wrote it."
	}

	if path, err := cfg.IndexPathOrDefault(); err == nil {
		said.Facts = append(said.Facts, fact{"index", path})
	}
	if built, ok := debug.ReadBuildInfo(); ok {
		if version := built.Main.Version; version != "" && version != "(devel)" {
			said.Facts = append(said.Facts, fact{"this build", version})
		}
		for _, setting := range built.Settings {
			if setting.Key == "vcs.revision" {
				said.Facts = append(said.Facts, fact{"revision", setting.Value})
			}
		}
	}

	var out strings.Builder
	if err := refusalPage.Execute(&out, said); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

var refusalPage = template.Must(template.New("refusal").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>numen</title>
<style>
  :root { color-scheme: light dark; }
  body {
    margin: 0; padding: 3rem 3rem 2rem;
    font: 14px/1.5 ui-sans-serif, system-ui, sans-serif;
    color: light-dark(#1d1f22, #dfe2e6);
    background: light-dark(#fbfbfa, #17181a);
  }
  h1 { margin: 0 0 1rem; font-size: 1.2rem; font-weight: 600; }
  p { margin: 0 0 1.5rem; max-width: 46ch; }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 0.4rem 1.5rem; margin: 0; }
  dt { color: light-dark(#7b7f87, #8b9098); }
  dd { margin: 0; font-family: ui-monospace, monospace; overflow-wrap: anywhere; }
  .do { margin-top: 2rem; font-weight: 600; }
</style>
</head>
<body>
  <h1>numen cannot open this vault</h1>
  <p>{{ .Says }}</p>
  <dl>
  {{- range .Facts }}
    <dt>{{ .Name }}</dt><dd>{{ .Value }}</dd>
  {{- end }}
  </dl>
  {{- if .Do }}
  <p class="do">{{ .Do }}</p>
  {{- end }}
</body>
</html>
`))
