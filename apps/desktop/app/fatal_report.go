package app

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

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
)

// failWithDialog shows what stopped the application, in a window.
//
// A person who opened numen from a dock has no terminal, and a message written
// to one is a window that never appeared. Whatever this build could not do, it
// says here, with what it knows about the state it found.
func failWithDialog(cfg container.Config, why error) {
	page, err := fatalReport{}.page(cfg, why)
	if err != nil {
		return
	}

	app := application.New(application.Options{
		Name: appName,
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(page)
			}),
		},
		// The application ends when its last window closes, and an error ends it
		// with the status the failure it draws has.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		PostShutdown: func() { os.Exit(1) },
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  appName,
		Width:  720,
		Height: 520,
		URL:    "/",
	})
	_ = app.Run()
}

// fatalReport is the start failure page: a heading, a one-line reason, the
// paths involved, the next step, and the raw error behind a disclosure.
type fatalReport struct {
	Heading  string
	Sentence string
	Facts    []detail
	Remedy   string
	// Error is the underlying error text, shown on request. Empty where the
	// sentence already says everything.
	Error string
}

type detail struct{ Name, Content string }

// Page headings. Each names what failed: the index, the settings file, or the start itself.
const (
	openingTheIndex    = "Numen can't open its index"
	readingTheSettings = "Numen can't read its settings"
	startingAtAll      = "Numen can't start"
)

const factSettings = "settings"

// page is the report as one self-contained document; a build that failed to start cannot serve its own assets.
func (fatalReport) page(cfg container.Config, why error) ([]byte, error) {
	response := getFatalReport(cfg, why)

	if constructed, ok := debug.ReadBuildInfo(); ok {
		if version := constructed.Main.Version; version != "" && version != "(devel)" {
			response.Facts = append(response.Facts, detail{"this build", version})
		}
		for _, setting := range constructed.Settings {
			if setting.Key == "vcs.revision" {
				response.Facts = append(response.Facts, detail{"revision", setting.Value})
			}
		}
	}

	var out strings.Builder
	if err := fatalPage.Execute(&out, response); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

// getFatalReport maps a start error to the page: what failed, the paths involved, and the next step.
func getFatalReport(cfg container.Config, why error) fatalReport {
	var outside *settings.OutsideBounds
	if errors.As(why, &outside) {
		return getSizeFatalReport(cfg, outside)
	}

	// sqlite says what it could not do as a result code, and the sentence it
	// carries beside it is the library's own.
	var coded interface{ Code() int }
	if errors.As(why, &coded) {
		return getIndexFatalReport(cfg, coded.Code(), why)
	}

	var unknown *vault.UnknownError
	if errors.As(why, &unknown) {
		return fatalReport{
			Heading:  startingAtAll,
			Sentence: "Vault not found: " + unknown.Name,
			Facts:    []detail{{"vault", unknown.Name}, {factSettings, getSettingsPath(cfg)}},
			Remedy:   "Start Numen without -vault to choose a vault from the list.",
		}
	}

	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(why, &syntax) || errors.As(why, &typeErr) {
		return fatalReport{
			Heading:  readingTheSettings,
			Sentence: "The settings file is not valid JSON.",
			Facts:    []detail{{factSettings, getSettingsPath(cfg)}},
			Remedy:   "Fix the file, or move it aside and Numen creates a new one.",
			Error:    why.Error(),
		}
	}

	return fatalReport{
		Heading:  startingAtAll,
		Sentence: "An unexpected error stopped the start.",
		Facts:    []detail{{"index", getIndexPath(cfg)}, {factSettings, getSettingsPath(cfg)}},
		Remedy:   "Start Numen again. If this page comes back, report the details below.",
		Error:    why.Error(),
	}
}

// getSizeFatalReport is a scale value out of range, from the file or the command line.
func getSizeFatalReport(cfg container.Config, outside *settings.OutsideBounds) fatalReport {
	far := fmt.Sprintf("%v to %v", outside.Least, outside.Most)
	content := fmt.Sprint(outside.Number)

	if strings.HasPrefix(outside.At, "-") {
		return fatalReport{
			Heading:  startingAtAll,
			Sentence: fmt.Sprintf("%s is out of range.", outside.At),
			Facts: []detail{
				{"flag", outside.At},
				{"value", content},
				{"range", far},
			},
			Remedy: fmt.Sprintf("Give %s a number from %s, or leave it out.", outside.At, far),
		}
	}
	return fatalReport{
		Heading:  readingTheSettings,
		Sentence: fmt.Sprintf("%s is out of range.", outside.At),
		Facts: []detail{
			{factSettings, getSettingsPath(cfg)},
			{"field", outside.At},
			{"value", content},
			{"range", far},
		},
		Remedy: fmt.Sprintf("Write a number from %s in %s, or remove the field to use 1.",
			far, outside.At),
	}
}

// The result codes sqlite answers with: a file that is not a database, and a
// file it could neither open nor make.
const (
	notADatabase = 26
	cannotOpen   = 14
)

// getIndexFatalReport is the index failing to open, one report per way a path can fail.
func getIndexFatalReport(cfg container.Config, code int, why error) fatalReport {
	pos := getIndexPath(cfg)
	response := fatalReport{Heading: openingTheIndex, Facts: []detail{{"index", pos}}}

	folder := false
	if info, err := os.Stat(pos); err == nil {
		folder = info.IsDir()
	}

	switch {
	case code == notADatabase:
		response.Sentence = "This file is not a Numen index."
		response.Remedy = "Move it aside. The index is a cache; Numen rebuilds it from your vaults."
	case code == cannotOpen && folder:
		response.Sentence = "The index path is a folder."
		response.Remedy = "Point -index at a file, or move the folder out of the way."
	case code == cannotOpen:
		response.Sentence = "The index can't be opened or created here."
		response.Facts = append(response.Facts, detail{"folder", filepath.Dir(pos)})
		response.Remedy = "Allow writing in the folder (check its permission), " +
			"or point -index at a writable path."
	default:
		response.Sentence = "The index can't be opened."
		response.Remedy = "Point -index at another path. If that fails too, report the details below."
		response.Error = why.Error()
	}
	return response
}

// getIndexPath is the file the index is kept in, whether or not one was named.
func getIndexPath(cfg container.Config) string {
	path, err := cfg.IndexPathOrDefault()
	if err != nil {
		return cfg.IndexPath
	}
	return path
}

// getSettingsPath is the file a person configures this installation in. A registry
// pointed somewhere chosen takes the settings with it.
func getSettingsPath(cfg container.Config) string {
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

var fatalPage = template.Must(template.New("fatal").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Numen</title>
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
  details { margin-top: 1.5rem; color: light-dark(#7b7f87, #8b9098); }
  summary { cursor: pointer; }
  pre { margin: 0.5rem 0 0; font-family: ui-monospace, monospace; white-space: pre-wrap; overflow-wrap: anywhere; }
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
  {{- if .Error }}
  <details><summary>Details</summary><pre>{{ .Error }}</pre></details>
  {{- end }}
</body>
</html>
`))
