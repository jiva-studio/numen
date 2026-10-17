package plugincli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// CreateOptions declares the configuration for plugin scaffolding.
type CreateOptions struct {
	Name      string
	Type      string
	Template  string
	OutputDir string
}

// Create generates a complete plugin scaffold on disk.
func Create(opts CreateOptions, stdout io.Writer) error {
	rawName := strings.TrimSpace(opts.Name)
	if rawName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	shortName, id, title := parsePluginIdentity(rawName)
	targetDir := resolveOutputDir(opts.OutputDir, shortName)

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create plugin directory: %w", err)
	}

	hasBackend, hasFrontend := determineComponents(opts.Type, opts.Template)
	if err := writeManifest(targetDir, id, title, shortName, opts.Template, hasBackend, hasFrontend); err != nil {
		return err
	}

	if hasBackend {
		if err := writeBackendFiles(targetDir, shortName, opts.Template); err != nil {
			return err
		}
	}

	if hasFrontend {
		if err := writeFrontendFiles(targetDir, shortName, title); err != nil {
			return err
		}
	}

	if err := writeSupportFiles(targetDir, shortName, title, hasBackend); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Created plugin %s in %s\n", id, targetDir)
	return nil
}

func parsePluginIdentity(rawName string) (shortName, id, title string) {
	if strings.Contains(rawName, ".") {
		id = rawName
		parts := strings.Split(rawName, ".")
		shortName = parts[len(parts)-1]
	} else {
		shortName = strings.ToLower(rawName)
		id = "md.numen." + shortName
	}
	title = toTitleCase(shortName)
	return shortName, id, title
}

func resolveOutputDir(customDir, shortName string) string {
	if customDir != "" {
		return customDir
	}
	for _, cand := range []string{"modules/plugins", "../../plugins", "../plugins"} {
		if info, err := os.Stat(cand); err == nil && info.IsDir() {
			return filepath.Join(cand, shortName)
		}
	}
	return shortName
}

func determineComponents(pluginType, template string) (hasBackend, hasFrontend bool) {
	switch template {
	case "reader":
		return true, true
	case "compute":
		return true, false
	case "agent":
		return true, true
	default:
		if pluginType == "frontend" {
			return false, true
		}
		return true, false
	}
}

func writeManifest(targetDir, id, title, shortName, template string, hasBackend, hasFrontend bool) error {
	capabilities, events := getCapabilitiesAndEvents(template, shortName, hasBackend, hasFrontend)
	keepAlive := template == "reader"

	manifest := port.PluginManifest{
		ID:               id,
		Name:             title,
		Version:          "0.1.0",
		Protocol:         port.PluginProtocolVersion,
		Description:      fmt.Sprintf("%s plugin for Numen", title),
		Capabilities:     capabilities,
		Targets:          []string{"desktop.editor"},
		ActivationEvents: events,
		Permissions:      []string{"vault:read"},
		Lifecycle: port.PluginLifecycle{
			IdleTimeoutSeconds:    180,
			KeepAliveWhileVisible: keepAlive,
		},
		Settings: []port.PluginManifestSetting{
			{
				Key:         "enabled",
				Type:        "boolean",
				Title:       "Enable Plugin",
				Description: fmt.Sprintf("Enable or disable %s features", title),
				Default:     true,
			},
		},
	}

	if hasBackend {
		manifest.Backend = port.PluginBackend{
			Type:        "process",
			Entrypoint:  "bin/" + shortName,
			ServiceName: fmt.Sprintf("numen.v1.%sService", title),
		}
	}

	if hasFrontend {
		manifest.Frontend = port.PluginFrontend{
			Entrypoint: "dist/index.js",
			Styles:     "dist/styles.css",
		}
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(targetDir, "manifest.json"), append(data, '\n'), 0o644)
}

func getCapabilitiesAndEvents(template, shortName string, hasBackend, hasFrontend bool) ([]string, []string) {
	switch template {
	case "reader":
		return []string{"document"}, []string{fmt.Sprintf("onFile:*.%s", shortName)}
	case "compute":
		return []string{"compute"}, []string{fmt.Sprintf("onCommand:%s.compute", shortName)}
	case "agent":
		return []string{"agent"}, []string{fmt.Sprintf("onCommand:%s.chat", shortName)}
	default:
		if hasFrontend && !hasBackend {
			return []string{"ui"}, []string{fmt.Sprintf("onCommand:%s.open", shortName)}
		}
		return []string{"service"}, []string{fmt.Sprintf("onCommand:%s.run", shortName)}
	}
}

func writeBackendFiles(targetDir, shortName, template string) error {
	backendDir := filepath.Join(targetDir, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		return err
	}

	goModContent := fmt.Sprintf(`module github.com/jiva-studio/numen/modules/plugins/%s

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	golang.org/x/net v0.58.0
)

require (
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go
replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
`, shortName)

	if err := os.WriteFile(filepath.Join(targetDir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		return err
	}

	goSumContent := `connectrpc.com/connect v1.21.0 h1:LhqSJt7jHf5NJBo9Jq/t/9FjcYAideif0mg+qe2jCUs=
connectrpc.com/connect v1.21.0/go.mod h1:A2ygJrukXwWy32vkCAAHNVguZrqZ+jeZ9rGRnGR4dN4=
github.com/google/go-cmp v0.7.0 h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=
github.com/google/go-cmp v0.7.0/go.mod h1:pXiqmnSA92OHEEa9HXL2W4E7lf9JzCmGVUdgjX3N/iU=
golang.org/x/net v0.58.0 h1:ynWG7rqYi4ccpTEuPZ2QGWHktVEM9DMCj9yzDE0Q7To=
golang.org/x/net v0.58.0/go.mod h1:YwCddHnFlT7eLQqVprV19OnhLGtc5xOKgE0RyqgfWAU=
golang.org/x/text v0.41.0 h1:vz/seA0lnX87Othu2f/0L24RcgrXD9/YFTSuGjj3rH8=
golang.org/x/text v0.41.0/go.mod h1:jvf1O8ajNzZqhSrQBPbutR/EB83Cc0CFrezNQIwbb5M=
google.golang.org/protobuf v1.36.12 h1:pJOKDDOyeXErUroCihFAd5LQuwXBSpVnKGrj5o/fwxc=
google.golang.org/protobuf v1.36.12/go.mod h1:HTf+CrKn2C3g5S8VImy6tdcUvCska2kB7j23XfzDpco=
`
	if err := os.WriteFile(filepath.Join(targetDir, "go.sum"), []byte(goSumContent), 0o644); err != nil {
		return err
	}

	var mainContent string
	if template == "reader" {
		mainContent = readerMainGoTemplate
	} else {
		mainContent = genericMainGoTemplate
	}

	return os.WriteFile(filepath.Join(backendDir, "main.go"), []byte(mainContent), 0o644)
}

const readerMainGoTemplate = `package main

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
)

type documentServer struct{}

func (s *documentServer) GetDocument(
	_ context.Context,
	_ *connect.Request[numenv1.GetDocumentRequest],
) (*connect.Response[numenv1.GetDocumentResponse], error) {
	return connect.NewResponse(&numenv1.GetDocumentResponse{
		Pages: []*numenv1.Page{
			{Width: 800, Height: 600},
		},
	}), nil
}

func (s *documentServer) RenderPage(
	_ context.Context,
	_ *connect.Request[numenv1.RenderPageRequest],
) (*connect.Response[numenv1.RenderPageResponse], error) {
	return connect.NewResponse(&numenv1.RenderPageResponse{
		ImageData:   nil,
		ContentType: "image/jpeg",
		Width:       800,
		Height:      600,
	}), nil
}

func (s *documentServer) ExtractText(
	_ context.Context,
	_ *connect.Request[numenv1.ExtractTextRequest],
) (*connect.Response[numenv1.ExtractTextResponse], error) {
	return connect.NewResponse(&numenv1.ExtractTextResponse{
		Text:       "Scaffolded document text.",
		PageStarts: []int32{0},
	}), nil
}

func (s *documentServer) GetHighlights(
	_ context.Context,
	_ *connect.Request[numenv1.GetHighlightsRequest],
) (*connect.Response[numenv1.GetHighlightsResponse], error) {
	return connect.NewResponse(&numenv1.GetHighlightsResponse{
		Highlights: nil,
	}), nil
}

func main() {
	s := server.New()
	s.Register(numenv1connect.NewDocumentServiceHandler(&documentServer{}, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
`

const genericMainGoTemplate = `package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
)

func main() {
	s := server.New()

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
`

func writeFrontendFiles(targetDir, shortName, title string) error {
	frontendDir := filepath.Join(targetDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		return err
	}
	if err := writeFrontendCodeFiles(frontendDir, shortName, title); err != nil {
		return err
	}
	return writeFrontendConfigFiles(frontendDir, shortName)
}

func writeFrontendCodeFiles(frontendDir, shortName, title string) error {
	indexTS := fmt.Sprintf(`import type { PluginContext } from '@numen/plugin-sdk'

// activate is called when the plugin matches an activation event.
export function activate(ctx: PluginContext): void {
  ctx.registerCommand({
    id: '%s.action',
    title: '%s: Run Action',
    handler: () => {
      // Handle action
    },
  })
}

// deactivate is called when the plugin is unloaded.
export function deactivate(): void {
  // Teardown state
}
`, shortName, title)

	viewerVue := fmt.Sprintf(`<template>
  <div class="plugin-view">
    <h2>{{ title }}</h2>
  </div>
</template>

<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
defineProps<{
  filePath?: string
}>()

/* --------------------------------- State ---------------------------------- */
const title = '%s'
</script>

<style scoped>
.plugin-view {
  padding: 1.5rem;
}
</style>
`, title)

	stylesCSS := `.plugin-view {
  display: block;
}
`

	if err := os.WriteFile(filepath.Join(frontendDir, "index.ts"), []byte(indexTS), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "Viewer.vue"), []byte(viewerVue), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(frontendDir, "styles.css"), []byte(stylesCSS), 0o644)
}

func writeFrontendConfigFiles(frontendDir, shortName string) error {
	pkgJSON := fmt.Sprintf(`{
  "name": "@numen/plugin-%s",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "vite build",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@numen/plugin-sdk": "*",
    "vue": "^3.5.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^6.0.0",
    "typescript": "^6.0.3",
    "vite": "^8.3.2"
  }
}
`, shortName)

	viteConfig := `import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    lib: {
      entry: 'index.ts',
      formats: ['es'],
      fileName: () => 'index.js',
    },
    outDir: '../dist',
    emptyOutDir: true,
  },
})
`

	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(frontendDir, "vite.config.ts"), []byte(viteConfig), 0o644)
}

func writeSupportFiles(targetDir, shortName, title string, hasBackend bool) error {
	var makefileContent string
	if hasBackend {
		makefileContent = fmt.Sprintf(`.PHONY: build test

build:
	mkdir -p bin
	go build -o bin/%s ./backend

test:
	go test ./backend/...
`, shortName)
	} else {
		makefileContent = `.PHONY: build test

build:
	cd frontend && npm run build

test:
	cd frontend && npm run typecheck
`
	}

	if err := os.WriteFile(filepath.Join(targetDir, "Makefile"), []byte(makefileContent), 0o644); err != nil {
		return err
	}

	readmeContent := fmt.Sprintf(`# %s Plugin

Numen modular plugin scaffolded with `+"`numen plugin create`"+`.

## Structure

- `+"`manifest.json`"+`: plugin metadata, capabilities, permissions, and lifecycle rules.
- `+"`backend/`"+`: Connect-RPC Go server implementation.
- `+"`frontend/`"+`: Vue 3 / TypeScript client interface.

## Building

`+"```bash"+`
make build
`+"```"+`
`, title)

	return os.WriteFile(filepath.Join(targetDir, "README.md"), []byte(readmeContent), 0o644)
}

func toTitleCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}
