package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
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
		// The application ends when its last window closes, and a refusal ends it
		// with the status the failure it draws has.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		PostShutdown: func() { os.Exit(1) },
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen",
		Width:  720,
		Height: 520,
		URL:    "/",
	})
	_ = app.Run()
}

// refusal is what the page says: what could not be opened, the sentence, the
// facts behind it, and what to do about it.
type refusal struct {
	Heading  string
	Sentence string
	Facts    []fact
	Remedy   string
}

type fact struct{ Name, Content string }

// The headings a refusal is drawn under. Each names the thing that could not be
// opened, which is the index or the person's own settings file.
const (
	openingTheIndex    = "numen cannot open its index"
	readingTheSettings = "numen cannot read its settings"
	startingAtAll      = "numen cannot start"
)

// page is the refusal as one document. Everything it needs is inside it: a
// build that could not open its own index cannot serve its own interface
// either.
func (refusal) page(cfg container.Config, why error) ([]byte, error) {
	said := stopped(cfg, why)

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

// stopped is the state the application is in, said in its own words: what it
// could not open, what it found, and what a person can do about it.
func stopped(cfg container.Config, why error) refusal {
	var outside *settings.OutsideBounds
	if errors.As(why, &outside) {
		return sized(cfg, outside)
	}

	// sqlite says what it could not do as a result code, and the sentence it
	// carries beside it is the library's own.
	var coded interface{ Code() int }
	if errors.As(why, &coded) {
		return indexing(cfg, coded.Code(), why)
	}

	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	if errors.As(why, &syntax) || errors.As(why, &typed) {
		return refusal{
			Heading:  readingTheSettings,
			Sentence: "This settings file is not JSON, so numen cannot tell what it was asked for.",
			Facts:    []fact{{"settings", settingsAt(cfg)}, {"what was read", why.Error()}},
			Remedy: "Put the file right, or move it aside: " +
				"numen writes a new one holding what it is doing.",
		}
	}

	return refusal{
		Heading:  startingAtAll,
		Sentence: why.Error(),
		Facts:    []fact{{"index", indexAt(cfg)}, {"settings", settingsAt(cfg)}},
		Remedy:   "Start numen again, and report what this window says if it stops here every time.",
	}
}

// sized is a number a size does not take, from the file or from the command
// line.
func sized(cfg container.Config, outside *settings.OutsideBounds) refusal {
	far := fmt.Sprintf("%v to %v", outside.Least, outside.Most)
	written := fmt.Sprint(outside.Number)

	if strings.HasPrefix(outside.At, "-") {
		return refusal{
			Heading:  startingAtAll,
			Sentence: fmt.Sprintf("%s was given a number the size does not take.", outside.At),
			Facts: []fact{
				{"on the command line", outside.At},
				{"given", written},
				{"as far as the size goes", far},
			},
			Remedy: fmt.Sprintf("Give %s a number from %s, or leave it out.", outside.At, far),
		}
	}
	return refusal{
		Heading:  readingTheSettings,
		Sentence: fmt.Sprintf("%s is a number the size does not take.", outside.At),
		Facts: []fact{
			{"settings", settingsAt(cfg)},
			{"field", outside.At},
			{"written", written},
			{"as far as the size goes", far},
		},
		Remedy: fmt.Sprintf("Write a number from %s in %s, or take the field out to run at 1.",
			far, outside.At),
	}
}

// The result codes sqlite answers with: a file that is not a database, and a
// file it could neither open nor make.
const (
	notADatabase = 26
	cannotOpen   = 14
)

// indexing is the index refusing to open, which is one fault for each way a
// path can fail to be an index.
func indexing(cfg container.Config, code int, why error) refusal {
	at := indexAt(cfg)
	said := refusal{Heading: openingTheIndex, Facts: []fact{{"index", at}}}

	folder := false
	if info, err := os.Stat(at); err == nil {
		folder = info.IsDir()
	}

	switch {
	case code == notADatabase:
		said.Sentence = "This file is not a numen index."
		said.Remedy = "Move it aside. The index is a cache: numen makes a new one " +
			"and fills it from your vaults."
	case code == cannotOpen && folder:
		said.Sentence = "An index is a file, and this path is a folder."
		said.Remedy = "Point -index at a file, or move the folder out of the way."
	case code == cannotOpen:
		said.Sentence = "numen could neither open an index here nor make one."
		said.Facts = append(said.Facts, fact{"folder", filepath.Dir(at)})
		said.Remedy = "Give yourself permission to write in the folder, " +
			"or point -index somewhere you can write."
	default:
		said.Sentence = why.Error()
		said.Remedy = "Point -index at another path, and report what this window says."
	}
	return said
}

// indexAt is the file the index is kept in, whether or not one was named.
func indexAt(cfg container.Config) string {
	path, err := cfg.IndexPathOrDefault()
	if err != nil {
		return cfg.IndexPath
	}
	return path
}

// settingsAt is the file a person configures this installation in. A registry
// pointed somewhere chosen takes the settings with it.
func settingsAt(cfg container.Config) string {
	if cfg.SettingsPath != "" {
		return cfg.SettingsPath
	}
	if cfg.RegistryPath != "" {
		return filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	}
	path, err := settings.Path()
	if err != nil {
		return ""
	}
	return path
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
  <h1>{{ .Heading }}</h1>
  <p>{{ .Sentence }}</p>
  <dl>
  {{- range .Facts }}
    <dt>{{ .Name }}</dt><dd>{{ .Content }}</dd>
  {{- end }}
  </dl>
  {{- if .Remedy }}
  <p class="do">{{ .Remedy }}</p>
  {{- end }}
</body>
</html>
`))
