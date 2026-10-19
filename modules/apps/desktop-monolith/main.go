// Command main is the monolithic entry point: desktop application with plugins wired in-process.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/app"
	"github.com/jiva-studio/numen/modules/plugins/agent"
	agentacp "github.com/jiva-studio/numen/modules/plugins/agent-acp"
	acpservice "github.com/jiva-studio/numen/modules/plugins/agent-acp/backend/service"
	agentantigravity "github.com/jiva-studio/numen/modules/plugins/agent-antigravity"
	antigravityservice "github.com/jiva-studio/numen/modules/plugins/agent-antigravity/backend/service"
	agentclaude "github.com/jiva-studio/numen/modules/plugins/agent-claude"
	claudeservice "github.com/jiva-studio/numen/modules/plugins/agent-claude/backend/service"
	agentcodex "github.com/jiva-studio/numen/modules/plugins/agent-codex"
	codexservice "github.com/jiva-studio/numen/modules/plugins/agent-codex/backend/service"
	agentservice "github.com/jiva-studio/numen/modules/plugins/agent/backend/service"
	"github.com/jiva-studio/numen/modules/plugins/downloader"
	downloadservice "github.com/jiva-studio/numen/modules/plugins/downloader/backend/service"
	"github.com/jiva-studio/numen/modules/plugins/embeddings"
	embeddingservice "github.com/jiva-studio/numen/modules/plugins/embeddings/backend/service"
	"github.com/jiva-studio/numen/modules/plugins/ocr"
	ocrservice "github.com/jiva-studio/numen/modules/plugins/ocr/backend/service"
	"github.com/jiva-studio/numen/modules/plugins/proofreading"
	proofreadservice "github.com/jiva-studio/numen/modules/plugins/proofreading/backend/service"
	readerepub "github.com/jiva-studio/numen/modules/plugins/reader-epub"
	epubservice "github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
	readerpdf "github.com/jiva-studio/numen/modules/plugins/reader-pdf"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
	"github.com/jiva-studio/numen/modules/plugins/transcription"
	transcriptionservice "github.com/jiva-studio/numen/modules/plugins/transcription/backend/service"
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
