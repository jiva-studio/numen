//go:build unix

package bind

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDiag(t *testing.T) {
	keepEnvironment(t)
	d1 := t.TempDir()
	_ = os.Mkdir(filepath.Join(d1, "index.db"), 0o755)
	_, e := Start(d1)
	t.Logf("INDEX %T %v", e, e)
	d2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(d2, "vaults.json"), []byte("{not json"), 0o600)
	_, e = Start(d2)
	t.Logf("LIST %T %v", e, e)
	d3 := t.TempDir()
	_ = os.Mkdir(filepath.Join(d3, "vault"), 0o000)
	_, e = Start(d3)
	t.Logf("UNREADABLE %T %v", e, e)
	_ = os.Chmod(filepath.Join(d3, "vault"), 0o700)
	d4 := t.TempDir()
	f := filepath.Join(d4, "f")
	_ = os.WriteFile(f, nil, 0o600)
	_, e = Start(f)
	t.Logf("FILE %T %v", e, e)
	d5 := t.TempDir()
	_ = os.Chmod(d5, 0o500)
	_, e = registerVault(makeConfig(d5, io.Discard), t.TempDir())
	t.Logf("ADD %T %v", e, e)
	_ = os.Chmod(d5, 0o700)
}
