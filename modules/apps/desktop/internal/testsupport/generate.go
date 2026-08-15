package testsupport

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// GenerateVault writes a vault of n notes and gives it an identity.
//
// The content is synthetic but not uniform: notes differ in length, sit at
// several depths, and share a vocabulary so that a search has something to rank.
// A vault of identical notes would measure the parser against one shape and the
// full-text index against one term.
func GenerateVault(tb testing.TB, n int) domain.Vault {
	tb.Helper()
	root := tb.TempDir()

	// Deterministic: a benchmark that generates different work each run cannot
	// be compared with itself.
	r := rand.New(rand.NewPCG(1, 2))

	for i := range n {
		dir := filepath.Join(root, fmt.Sprintf("%02d", i%50))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
		path := filepath.Join(dir, fmt.Sprintf("note-%06d.md", i))
		if err := os.WriteFile(path, []byte(note(r, i)), 0o644); err != nil {
			tb.Fatal(err)
		}
	}

	cfg, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now())
	if err != nil {
		tb.Fatal(err)
	}
	return domain.Vault{ID: cfg.ID, Name: "generated", Path: root}
}

var vocabulary = strings.Fields(`entropy thermodynamics shannon information measure
	system description observer probability distribution ensemble equilibrium
	temperature energy conservation reversible irreversible statistical mechanics`)

func note(r *rand.Rand, i int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: Note %d\nstatus: generated\n---\n\n# Note %d\n\n", i, i)

	// Between roughly 40 and 400 words, in a few sections: real vaults are not
	// made of one note repeated.
	sections := 1 + r.IntN(4)
	for s := range sections {
		fmt.Fprintf(&b, "## Section %d\n\n", s+1)
		for range 10 + r.IntN(90) {
			b.WriteString(vocabulary[r.IntN(len(vocabulary))])
			b.WriteByte(' ')
		}
		b.WriteString("\n\n")
	}
	return b.String()
}
