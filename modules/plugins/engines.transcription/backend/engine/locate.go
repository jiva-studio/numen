package engine

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
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
	from      string
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
	fileName := defaultFileName
	if base := filepath.Base(repoAddress); base != "" && base != "." && base != "/" && !strings.HasSuffix(repoAddress, "/") {
		fileName = base
	}
	if dir != "" {
		candidate := filepath.Join(dir, fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if home := os.Getenv("NUMEN_MODELS_DIR"); home != "" {
		candidate := filepath.Join(home, fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if base, err := os.UserConfigDir(); err == nil {
		candidate := filepath.Join(base, "numen", "models", fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
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
		from:      "provisioned",
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return engine, found, nil
}

func getModelName(name, pos string) string {
	if name != "" {
		return name
	}
	return strings.TrimSuffix(path.Base(filepath.ToSlash(pos)), filepath.Ext(pos))
}
