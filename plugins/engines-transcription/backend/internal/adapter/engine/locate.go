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
	fileName := resolveFileName(repoAddress, defaultFileName)
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found", fileName)
}

// FetchDependency resolves a model that the manifest declares under an id through the host.
type FetchDependency func(ctx context.Context, id string) (string, error)

type modelFiles struct {
	dir            string
	shouldDownload bool
	fetch          FetchDependency
}

func (files modelFiles) resolve(ctx context.Context, path, address, name, dependency string) (string, error) {
	local, err := locateModel(path, address, name, files.dir)
	if err == nil || path != "" || dependency == "" || !files.shouldDownload {
		return local, err
	}
	return files.fetch(ctx, dependency)
}

func locateModels(ctx context.Context, library *onnx.Library, cfg settings.Config, fetch FetchDependency) (*ort.Engine, paths, error) {
	opened, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, paths{}, err
	}
	found, err := modelFiles{dir: cfg.Dir, shouldDownload: cfg.ShouldDownload, fetch: fetch}.resolveAll(ctx, cfg)
	if err != nil {
		return nil, paths{}, err
	}
	return opened.Engine, found, nil
}

func (files modelFiles) resolveAll(ctx context.Context, cfg settings.Config) (paths, error) {
	encoder, err := files.resolve(ctx, cfg.Model.Encoder, cfg.Model.Repo, encoderFile, cfg.Model.Dependencies.Encoder)
	if err != nil {
		return paths{}, fmt.Errorf("encoder model: %w", err)
	}
	decoder, err := files.resolve(ctx, cfg.Model.Decoder, cfg.Model.Repo, decoderFile, cfg.Model.Dependencies.Decoder)
	if err != nil {
		return paths{}, fmt.Errorf("decoder model: %w", err)
	}
	joiner, err := files.resolve(ctx, cfg.Model.Joiner, cfg.Model.Repo, joinerFile, cfg.Model.Dependencies.Joiner)
	if err != nil {
		return paths{}, fmt.Errorf("joiner model: %w", err)
	}
	tokens, err := files.resolve(ctx, cfg.Model.Tokens, cfg.Model.Repo, tokensFile, cfg.Model.Dependencies.Tokens)
	if err != nil {
		return paths{}, fmt.Errorf("tokens file: %w", err)
	}
	segmenterPath, err := files.resolve(ctx, cfg.Segmenter.Path, cfg.Segmenter.Repo, "silero.onnx", cfg.Segmenter.Dependency)
	if err != nil {
		return paths{}, fmt.Errorf("segmenter model: %w", err)
	}
	return paths{encoder: encoder, decoder: decoder, joiner: joiner, tokens: tokens, segmenter: segmenterPath}, nil
}

func getCacheRoot() string {
	root, err := cache.GetSharedDir()
	if err != nil {
		return ""
	}
	return root
}
