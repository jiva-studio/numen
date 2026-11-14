package onnx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readyName is the file the host writes beside a model it provisioned. Its first line names the
// model file and its second the SHA-256 the model was verified against.
const readyName = ".ready"

func getModelKey(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if sum, ok := readProvisionedSum(path); ok {
		return sum, nil
	}
	hash := sha256.New()
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if _, err := fmt.Fprintf(hash, "%s\n%d\n%d", filepath.Base(path), info.Size(), info.ModTime().UnixNano()); err != nil {
		return "", err
	}
	return "stat-" + hex.EncodeToString(hash.Sum(nil)), nil
}

func readProvisionedSum(path string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), readyName))
	if err != nil {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != filepath.Base(path) || !isDigest(strings.TrimSpace(lines[1])) {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(lines[1])), true
}

func isDigest(text string) bool {
	if len(text) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(text)
	return err == nil
}
