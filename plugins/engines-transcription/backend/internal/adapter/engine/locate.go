package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
)

// paths are the files this run transcribes with, and where they were found.
type paths struct {
	encoder   string
	decoder   string
	joiner    string
	tokens    string
	segmenter string
}

func findInFolder(dir, fileName string) string {
	if dir == "" || fileName == "" {
		return ""
	}
	candidate := filepath.Join(dir, fileName)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func resolveFileName(repoAddress, defaultFileName string) string {
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="filepath.Base never returns empty and only returns slash when repoAddress ends with slash"
	if base := filepath.Base(repoAddress); base != "" && base != "." && base != "/" && !strings.HasSuffix(repoAddress, "/") {
		return base
	}
	return defaultFileName
}

func locateModel(explicitPath, repoAddress, defaultFileName, dir string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if repoAddress == "" {
		return "", fmt.Errorf("model %s is not named: specify path or repo", defaultFileName)
	}
	fileName := resolveFileName(repoAddress, defaultFileName)
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found", fileName)
}

// FetchModelFile resolves a pinned model through the host.
type FetchModelFile func(context.Context, string, string) (string, error)

type modelFiles struct {
	dir            string
	shouldDownload bool
	fetch          FetchModelFile
}

func (files modelFiles) resolve(ctx context.Context, path, address, name string) (string, error) {
	local, err := locateModel(path, address, name, files.dir)
	if strings.HasSuffix(address, "/") {
		address += name
	}
	if err == nil || path != "" || !files.shouldDownload {
		return local, err
	}
	digest, ok := settings.FindModelDigest(address)
	if !ok {
		return "", fmt.Errorf("no SHA-256 pinned for model %q", address)
	}
	return files.fetch(ctx, address, digest)
}

func locateModels(ctx context.Context, library *onnx.Library, cfg settings.Config, fetch FetchModelFile) (*ort.Engine, paths, error) {
	engine, _, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, paths{}, err
	}
	files := modelFiles{dir: cfg.Dir, shouldDownload: cfg.ShouldDownload, fetch: fetch}
	encoder, err := files.resolve(ctx, cfg.Model.Encoder, cfg.Model.Repo, encoderFile)
	if err != nil {
		return nil, paths{}, fmt.Errorf("encoder model: %w", err)
	}
	decoder, err := files.resolve(ctx, cfg.Model.Decoder, cfg.Model.Repo, decoderFile)
	if err != nil {
		return nil, paths{}, fmt.Errorf("decoder model: %w", err)
	}
	joiner, err := files.resolve(ctx, cfg.Model.Joiner, cfg.Model.Repo, joinerFile)
	if err != nil {
		return nil, paths{}, fmt.Errorf("joiner model: %w", err)
	}
	tokens, err := files.resolve(ctx, cfg.Model.Tokens, cfg.Model.Repo, tokensFile)
	if err != nil {
		return nil, paths{}, fmt.Errorf("tokens file: %w", err)
	}
	segmenterPath, err := files.resolve(ctx, cfg.Segmenter.Path, cfg.Segmenter.Repo, "silero.onnx")
	if err != nil {
		return nil, paths{}, fmt.Errorf("segmenter model: %w", err)
	}

	found := paths{
		encoder:   encoder,
		decoder:   decoder,
		joiner:    joiner,
		tokens:    tokens,
		segmenter: segmenterPath,
	}
	return engine, found, nil
}

func getCacheRoot() string {
	root, err := cache.GetSharedDir()
	if err != nil {
		return ""
	}
	return root
}
