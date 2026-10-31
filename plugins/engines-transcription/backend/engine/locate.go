package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// paths are the files this run transcribes with, and where they were found.
type paths struct {
	encoder   string
	decoder   string
	joiner    string
	tokens    string
	segmenter string
}

func resolveFileName(repoAddress, defaultFileName string) string {
	if base := filepath.Base(repoAddress); base != "" && base != "." && base != "/" && !strings.HasSuffix(repoAddress, "/") {
		return base
	}
	return defaultFileName
}

// locateModel finds a model file by checking:
// 1. Explicit path in settings
// 2. Provisioned environment variable (NUMEN_DEPENDENCY_*)
// 3. Plugin directory / models folder fallback (only if repo address is provided)
func locateModel(explicitPath, repoAddress, envVar, defaultFileName, dir string) (string, error) {
	configured, err := onnx.ResolveConfigured(explicitPath, envVar)
	if err != nil || configured != "" {
		return configured, err
	}
	if repoAddress == "" {
		return "", fmt.Errorf("model %s is not named: specify path or repo", defaultFileName)
	}
	return onnx.LocateInFolders(dir, resolveFileName(repoAddress, defaultFileName))
}

func locateModels(ctx context.Context, cfg Config) (*ort.Engine, paths, error) {
	engine, _, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, paths{}, err
	}
	encoder, err := locateModel(cfg.Model.Encoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, cfg.Dir)
	if err != nil {
		return nil, paths{}, fmt.Errorf("encoder model: %w", err)
	}
	decoder, err := locateModel(cfg.Model.Decoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_DECODER", decoderFile, cfg.Dir)
	if err != nil {
		return nil, paths{}, fmt.Errorf("decoder model: %w", err)
	}
	joiner, err := locateModel(cfg.Model.Joiner, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_JOINER", joinerFile, cfg.Dir)
	if err != nil {
		return nil, paths{}, fmt.Errorf("joiner model: %w", err)
	}
	tokens, err := locateModel(cfg.Model.Tokens, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_TOKENS", tokensFile, cfg.Dir)
	if err != nil {
		return nil, paths{}, fmt.Errorf("tokens file: %w", err)
	}
	segmenterPath, err := locateModel(cfg.Segmenter.Path, cfg.Segmenter.Repo, "NUMEN_DEPENDENCY_SILERO_VAD", "silero.onnx", cfg.Dir)
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
