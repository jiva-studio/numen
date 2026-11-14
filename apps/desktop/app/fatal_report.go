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

	"github.com/jiva-studio/numen/apps/desktop/internal/frame"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func failWithDialog(cfg container.Config, why error) error {
	page, err := fatalReport{}.page(cfg, why)
	if err != nil {
		return err
	}

	writeErrs := make(chan error, 1)

	app := application.New(application.Options{
		Name: appName,
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if _, err := w.Write(page); err != nil {
					select {
					case writeErrs <- err:
					default:
					}
				}
			}),
		},
		// The application ends when its last window closes, and an error ends it
		// with the status the failure it draws has.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.Window.NewWithOptions(getFatalWindowOptions())
	runErr := app.Run()
	select {
	case writeErr := <-writeErrs:
		return errors.Join(runErr, writeErr)
	default:
		return runErr
	}
}

func getFatalWindowOptions() application.WebviewWindowOptions {
	return frame.Window(application.WebviewWindowOptions{
		Title:  appName,
		Width:  720,
		Height: 520,
		URL:    "/",
	})
}

// fatalReport is the start failure page: a heading, a one-line reason, the
// paths involved, the next step, and the raw error behind a disclosure.
type fatalReport struct {
	Heading  string
	Sentence string
	Facts    []detail
	Remedy   string
	Error    string
}

type detail struct{ Name, Content string }

// Page headings. Each names what failed: the index, the settings file, or the start itself.
const (
	openingTheIndex    = "Numen can't open its index"
	readingTheSettings = "Numen can't read its settings"
	startingAtAll      = "Numen can't start"
)

const (
	factIndex    = "index"
	factSettings = "settings"
)

func (fatalReport) page(cfg container.Config, why error) ([]byte, error) {
	response := getFatalReport(cfg, why)

	if constructed, ok := debug.ReadBuildInfo(); ok {
		// gomutants:disable-next-line reason="a test binary has no module version, so the branch that reports one is reached only in a built release"
		if version := constructed.Main.Version; version != "" && version != "(devel)" {
			response.Facts = append(response.Facts, detail{"this build", version})
		}
		// gomutants:disable-next-line reason="a test binary carries no version-control settings, so the loop over them is reached only in a built release"
		for _, setting := range constructed.Settings {
			// gomutants:disable-next-line reason="a test binary carries no version-control settings, so the match is reached only in a built release"
			if setting.Key == "vcs.revision" {
				response.Facts = append(response.Facts, detail{"revision", setting.Value})
			}
		}
	}

	var out strings.Builder
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err := getFatalPage().Execute(&out, response); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

func getFatalReport(cfg container.Config, why error) fatalReport {
	if outside, ok := errors.AsType[*settingsdomain.OutsideBoundsError](why); ok {
		return getSizeFatalReport(cfg, outside)
	}

	if newer, ok := errors.AsType[*index.NewerIndexError](why); ok {
		return fatalReport{
			Heading:  openingTheIndex,
			Sentence: "The index was written by a newer Numen.",
			Facts: []detail{
				{factIndex, newer.File},
				{"index version", fmt.Sprint(newer.Found)},
				{"supported version", fmt.Sprint(newer.Supported)},
			},
			Remedy: "Update Numen.",
		}
	}

	// sqlite reports a failure as a result code, and the accompanying message
	// is the library's own.
	var coded interface{ Code() int }
	if errors.As(why, &coded) {
		return getIndexFatalReport(cfg, coded.Code(), why)
	}

	if name, ok := container.GetUnknownVaultName(why); ok {
		return fatalReport{
			Heading:  startingAtAll,
			Sentence: "Vault not found: " + name,
			Facts:    []detail{{"vault", name}, {factSettings, getSettingsPath(cfg)}},
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
		Facts:    []detail{{factIndex, getIndexPath(cfg)}, {factSettings, getSettingsPath(cfg)}},
		Remedy:   "Start Numen again. If this page comes back, report the details below.",
		Error:    why.Error(),
	}
}

func getSizeFatalReport(cfg container.Config, outside *settingsdomain.OutsideBoundsError) fatalReport {
	far := fmt.Sprintf("%v to %v", outside.Min, outside.Max)
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

func getIndexFatalReport(cfg container.Config, code int, why error) fatalReport {
	pos := getIndexPath(cfg)
	response := fatalReport{Heading: openingTheIndex, Facts: []detail{{factIndex, pos}}}

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

func getIndexPath(cfg container.Config) string { return cfg.IndexPath }

func getSettingsPath(cfg container.Config) string {
	if cfg.SettingsPath != "" {
		return cfg.SettingsPath
	}
	if cfg.RegistryPath != "" {
		return filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	}
	path, err := settings.Path()
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err != nil {
		return ""
	}
	return path
}

func getFatalPage() *template.Template {
	return template.Must(template.New("fatal").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Numen</title>
<style>
  :root { color-scheme: light dark; }
  body {
    margin: 0; padding: 4.5rem 3rem 2rem;
    font: 14px/1.5 ui-sans-serif, system-ui, sans-serif;
    color: light-dark(#1d1f22, #dfe2e6);
    background: light-dark(#fbfbfa, #17181a);
  }
  header {
    position: fixed; inset: 0 0 auto 0; height: 2rem; box-sizing: border-box;
    display: flex; align-items: center; justify-content: space-between; padding: 0 0 0 0.75rem;
    border-bottom: 1px solid light-dark(#dcdcd8, #2a2c30); user-select: none;
    background: light-dark(#f1f1ef, #1d1f22);
  }
  header.mac { padding-left: 72px; justify-content: flex-start; }
  header.mac button { display: none; }
  header button {
    height: 100%; width: 2.75rem; border: 0; background: none; color: inherit; font: inherit;
  }
  header button:hover { background: #e81123; color: #fff; }
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
  <header style="--wails-draggable: drag">
    <span>Numen</span>
    <button type="button" data-close aria-label="Close">&#x2715;</button>
  </header>
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
<script type="module">
  if (navigator.userAgent.includes('Mac')) document.querySelector('header').classList.add('mac')
  document.querySelector('[data-close]').addEventListener('click', async () => {
    const runtime = await import('/wails/runtime.js')
    await runtime.Window.Close()
  })
</script>
</body>
</html>
`))
}
