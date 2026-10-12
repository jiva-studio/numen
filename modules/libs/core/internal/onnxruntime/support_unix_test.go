//go:build linux

package onnxruntime

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ebitengine/purego"
)

const childModeKey = "NUMEN_ONNX_SUPPORT_CHILD"

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

func isCppRuntimeMapped() bool {
	maps, err := os.ReadFile("/proc/self/maps")
	return err == nil && strings.Contains(string(maps), "libstdc++")
}

// TestSupportChild is the body of the processes the tests below start. Each
// starts with nothing loaded, and says whether the C++ runtime is mapped at the
// end.
func TestSupportChild(t *testing.T) {
	mode := os.Getenv(childModeKey)
	if mode == "" {
		t.Skip("run by the tests that start it")
	}
	switch mode {
	case "probe":
		for _, at := range findInstalled("libstdc++.so", "*-gcc-*-lib") {
			if _, err := purego.Dlopen(at, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				fmt.Println("RESULT:mapped")
				return
			}
		}
		fmt.Println("RESULT:unmapped")
		return
	case "support":
		preloadCppRuntime()
	case "open":
		httpsClient.Transport = failingTransport{}
		_, _, _ = Open(t.Context(), Settings{Section: "indexing.recognition", Dir: t.TempDir(), ShouldDownload: true})
	}
	if isCppRuntimeMapped() {
		fmt.Println("RESULT:mapped")
		return
	}
	fmt.Println("RESULT:unmapped")
}

func runChild(t *testing.T, mode string) string {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSupportChild$")
	cmd.Env = append(os.Environ(), childModeKey+"="+mode, "HOME="+home, "XDG_CACHE_HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s child: %v\n%s", mode, err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if result, ok := strings.CutPrefix(line, "RESULT:"); ok {
			return result
		}
	}
	t.Fatalf("%s child answered nothing:\n%s", mode, out)
	return ""
}

// A build published for Linux names the C++ runtime and expects the system to
// hold it, so a machine that keeps the file elsewhere has it loaded globally.
func TestSupportLoadsTheCppRuntimeThisMachineKeeps(t *testing.T) {
	if runChild(t, "probe") != "mapped" {
		t.Skip("this machine keeps no C++ runtime that loads")
	}
	if got := runChild(t, "support"); got != "mapped" {
		t.Errorf("support left the C++ runtime %s", got)
	}
}

func TestOpenLoadsTheCppRuntimeBeforeLookingForTheLibrary(t *testing.T) {
	if runChild(t, "probe") != "mapped" {
		t.Skip("this machine keeps no C++ runtime that loads")
	}
	if got := runChild(t, "open"); got != "mapped" {
		t.Errorf("Open left the C++ runtime %s", got)
	}
}
