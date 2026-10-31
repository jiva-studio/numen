package onnx

import (
	"os"
	"path/filepath"
	"testing"
)

const modelEnvVar = "NUMEN_DEPENDENCY_TEST_MODEL"

func writeModel(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLocateModelResolvesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "custom-model.onnx")
	writeModel(t, explicit)

	found, err := LocateModel(explicit, modelEnvVar, "model.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
}

func TestLocateModelFailsOnMissingExplicitPath(t *testing.T) {
	if _, err := LocateModel(filepath.Join(t.TempDir(), "absent.onnx"), "", "model.onnx", ""); err == nil {
		t.Fatal("expected an error for a configured path that does not exist")
	}
}

func TestLocateModelResolvesEnvironmentVariable(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "provisioned-model.onnx")
	writeModel(t, envPath)
	t.Setenv(modelEnvVar, envPath)

	found, err := LocateModel("", modelEnvVar, "model.onnx", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != envPath {
		t.Errorf("got %s, want %s", found, envPath)
	}
}

func TestLocateModelResolvesFolderFallback(t *testing.T) {
	dir := t.TempDir()
	fallback := filepath.Join(dir, "model.onnx")
	writeModel(t, fallback)

	found, err := LocateModel("", modelEnvVar, "model.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallback {
		t.Errorf("got %s, want %s", found, fallback)
	}
}

func TestLocateModelResolvesTheModelsFolderOfTheEnvironment(t *testing.T) {
	home := t.TempDir()
	inHome := filepath.Join(home, "model.onnx")
	writeModel(t, inHome)
	t.Setenv(ModelsDirVar, home)

	found, err := LocateModel("", modelEnvVar, "model.onnx", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != inHome {
		t.Errorf("got %s, want %s", found, inHome)
	}
}

func TestLocateModelFailsWhenNotFound(t *testing.T) {
	t.Setenv(ModelsDirVar, t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if _, err := LocateModel("", modelEnvVar, "non-existent.onnx", t.TempDir()); err == nil {
		t.Fatal("expected an error for a model that exists nowhere")
	}
}

func TestLocateModelPrecedence(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.onnx")
	writeModel(t, explicit)
	envPath := filepath.Join(dir, "env.onnx")
	writeModel(t, envPath)
	t.Setenv(modelEnvVar, envPath)
	writeModel(t, filepath.Join(dir, "model.onnx"))

	got, err := LocateModel(explicit, modelEnvVar, "model.onnx", dir)
	if err != nil || got != explicit {
		t.Errorf("explicit precedence failed: got %s, err %v", got, err)
	}
	got, err = LocateModel("", modelEnvVar, "model.onnx", dir)
	if err != nil || got != envPath {
		t.Errorf("environment precedence failed: got %s, err %v", got, err)
	}
}

func TestResolveConfiguredNamesNothingForAnEmptyConfiguration(t *testing.T) {
	got, err := ResolveConfigured("", modelEnvVar)
	if err != nil || got != "" {
		t.Errorf("got %q, %v; want empty and no error", got, err)
	}
}
