// Command main is the monolithic entry point: desktop application with plugins wired in-process.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/app"
	agentacp "github.com/jiva-studio/numen/modules/plugins/agents-acp"
	acpservice "github.com/jiva-studio/numen/modules/plugins/agents-acp/backend/service"
	agentantigravity "github.com/jiva-studio/numen/modules/plugins/agents-antigravity"
	antigravityservice "github.com/jiva-studio/numen/modules/plugins/agents-antigravity/backend/service"
	agentclaude "github.com/jiva-studio/numen/modules/plugins/agents-claude"
	claudeservice "github.com/jiva-studio/numen/modules/plugins/agents-claude/backend/service"
	agentcodex "github.com/jiva-studio/numen/modules/plugins/agents-codex"
	codexservice "github.com/jiva-studio/numen/modules/plugins/agents-codex/backend/service"
	agent "github.com/jiva-studio/numen/modules/plugins/agents-numen"
	agentservice "github.com/jiva-studio/numen/modules/plugins/agents-numen/backend/service"
	embeddings "github.com/jiva-studio/numen/modules/plugins/engines-embeddings"
	embeddingservice "github.com/jiva-studio/numen/modules/plugins/engines-embeddings/backend/service"
	ocr "github.com/jiva-studio/numen/modules/plugins/engines-ocr"
	ocrservice "github.com/jiva-studio/numen/modules/plugins/engines-ocr/backend/service"
	transcription "github.com/jiva-studio/numen/modules/plugins/engines-transcription"
	transcriptionservice "github.com/jiva-studio/numen/modules/plugins/engines-transcription/backend/service"
	readerepub "github.com/jiva-studio/numen/modules/plugins/readers-epub"
	epubservice "github.com/jiva-studio/numen/modules/plugins/readers-epub/backend/service"
	readerpdf "github.com/jiva-studio/numen/modules/plugins/readers-pdf"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/readers-pdf/backend/service"
	downloader "github.com/jiva-studio/numen/modules/plugins/tools-media-importer"
	downloadservice "github.com/jiva-studio/numen/modules/plugins/tools-media-importer/backend/service"
	proofreading "github.com/jiva-studio/numen/modules/plugins/tools-proofreading"
	proofreadservice "github.com/jiva-studio/numen/modules/plugins/tools-proofreading/backend/service"
)

func main() {
	registerPlugins(app.RegisterPlugin)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

// serveFunc makes the handler of a plugin's backend from the variables a launched process of the
// plugin is given.
type serveFunc = func(vars map[string]string) (http.Handler, error)

// registerPlugins registers each plugin this binary links in, with its own manifest and backend.
func registerPlugins(register func(manifest []byte, serve serveFunc)) {
	register(agent.Manifest, agentservice.NewHandler)
	register(agentacp.Manifest, acpservice.NewHandler)
	register(agentantigravity.Manifest, antigravityservice.NewHandler)
	register(agentclaude.Manifest, claudeservice.NewHandler)
	register(agentcodex.Manifest, codexservice.NewHandler)
	register(downloader.Manifest, downloadservice.NewHandler)
	register(embeddings.Manifest, embeddingservice.NewHandler)
	register(ocr.Manifest, ocrservice.NewHandler)
	register(proofreading.Manifest, proofreadservice.NewHandler)
	register(readerepub.Manifest, epubservice.NewHandler)
	register(readerpdf.Manifest, pdfservice.NewHandler)
	register(transcription.Manifest, transcriptionservice.NewHandler)
}
