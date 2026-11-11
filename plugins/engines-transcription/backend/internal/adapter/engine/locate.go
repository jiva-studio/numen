package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/onnx"
)

// EnvDir names the folder plugins keep their models in. The host sets it, to the folder a user chose or
// to models in the shared data directory.
const EnvDir = "NUMEN_MODELS_DIR"

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

func findInHome(fileName string) string {
	if home := os.Getenv(EnvDir); home != "" {
		if found := findInFolder(home, fileName); found != "" {
			return found
		}
	}
	if base, err := cache.GetSharedDataDirOrPluginDataDir(); err == nil {
		return findInFolder(filepath.Join(base, "models"), fileName)
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

func locateModel(explicitPath, repoAddress, envVar, defaultFileName, dir string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if env := findInEnv(envVar); env != "" {
		return env, nil
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
	return "", fmt.Errorf("model %s not found", fileName)
}

func findInEnv(envVar string) string {
	// gomutants:disable-next-line BRANCH_IF reason="os.Getenv returns empty for an empty variable name"
	if envVar == "" {
		return ""
	}
	env := os.Getenv(envVar)
	// gomutants:disable-next-line BRANCH_IF reason="os.Stat fails on an empty path"
	if env == "" {
		return ""
	}
	if _, err := os.Stat(env); err != nil {
		return ""
	}
	return env
}

func locateModels(ctx context.Context, library *onnx.Library, cfg settings.Config) (*ort.Engine, paths, error) {
	engine, _, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
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
