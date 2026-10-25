package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// EnvDir names the folder plugins keep their models in.
const EnvDir = "NUMEN_MODELS_DIR"

// paths are the files this run listens through, and where they were found.
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

func findInHome(fileName string) string {
	if home := os.Getenv(EnvDir); home != "" {
		if found := findInFolder(home, fileName); found != "" {
			return found
		}
	}
	if base, err := os.UserConfigDir(); err == nil {
		return findInFolder(filepath.Join(base, "numen", "models"), fileName)
	}
	return ""
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
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if envVar != "" {
		if env := os.Getenv(envVar); env != "" {
			if _, err := os.Stat(env); err == nil {
				return env, nil
			}
		}
	}
	if repoAddress == "" {
		return "", fmt.Errorf("model %s is not named: specify path or repo", defaultFileName)
	}
	fileName := resolveFileName(repoAddress, defaultFileName)
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	if found := findInHome(fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found (env: %s)", fileName, envVar)
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
