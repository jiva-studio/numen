package provision

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestProvision_RejectsReservedFileNames(t *testing.T) {
	data := []byte("model-bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	for _, name := range []string{".ready", ".ready.tmp", ".tmp.abc"} {
		t.Run(name, func(t *testing.T) {
			provisioner := newTestProvisioner(t, false)
			provisioner.cfg.Client = server.Client()
			req := newModelRequirement(server.URL+"/w.bin", data)
			req.Metadata = map[string]string{"fileName": name}

			resolved, err := provisioner.Provision(t.Context(), req, nil)

			if err == nil {
				t.Errorf("the file name %q was accepted: %+v", name, resolved)
			}
		})
	}
}

func TestADiscoveredNameCannotLeaveTheSearchedFolders(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../secret", "se*", "/etc/passwd"} {
		t.Run(name, func(t *testing.T) {
			provisioner := newTestProvisioner(t, true)
			provisioner.cfg.Discovery = discovery{dirs: []string{bin}}
			req := manifest.DependencyRequirement{ID: "tool", Kind: manifest.DependencyKindBinary, Metadata: map[string]string{"binary": name}}

			resolved, err := provisioner.Provision(t.Context(), req, nil)

			if err == nil {
				t.Errorf("the name %q found %s", name, resolved.PrimaryPath)
			}
		})
	}
}

func TestProvision_CachedDependencyRequiresMatchingChecksum(t *testing.T) {
	data := []byte("model-bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	if _, err := provisioner.Provision(t.Context(), newModelRequirement(server.URL+"/w.bin", data), nil); err != nil {
		t.Fatal(err)
	}

	pinned := newModelRequirement(server.URL+"/w.bin", bytes.Repeat([]byte("other"), 3))
	_, err := provisioner.Provision(t.Context(), pinned, nil)

	if !errors.Is(err, ErrChecksum) {
		t.Errorf("a file fetched with another checksum was served: %v", err)
	}
}

func TestAMachineFileThatIsNotThePinnedBytesIsNotTheDependency(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte("not the pinned bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	provisioner := newTestProvisioner(t, true)
	provisioner.cfg.Discovery = discovery{dirs: []string{bin}}
	req := manifest.DependencyRequirement{
		ID: "ffmpeg", Kind: manifest.DependencyKindBinary, SourceURL: "https://example.invalid/ffmpeg", ChecksumSHA256: sumOf([]byte("pinned")),
	}

	resolved, err := provisioner.Provision(t.Context(), req, nil)

	if !errors.Is(err, ErrOffline) {
		t.Errorf("the machine's file was taken for the pinned one: %+v, %v", resolved, err)
	}
}
