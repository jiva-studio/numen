package testsupport

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GenerateVault writes a vault of n notes and gives it an identity.
//
// The content is synthetic but not uniform: notes differ in length, sit at
// several depths, and share a vocabulary so a search has something to rank.
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
		path := filepath.Join(dir, name(i)+".md")
		if err := os.WriteFile(path, []byte(note(r, i, n)), 0o644); err != nil {
			tb.Fatal(err)
		}
	}

	cfg, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now())
	if err != nil {
		tb.Fatal(err)
	}
	return domain.Vault{ID: domain.VaultID(cfg.ID), Name: "generated", Path: root}
}

// RareTerm appears in one generated note in a thousand.
const RareTerm = "hapaxlegomenon"

var vocabulary = strings.Fields(`entropy thermodynamics shannon information measure
	system description observer probability distribution ensemble equilibrium
	temperature energy conservation reversible irreversible statistical mechanics`)

func note(r *rand.Rand, i, total int) string {
	var b strings.Builder

	// Notes without links skip the link path and would make the parser and index
	// appear faster than they are. Every note names a parent and links to a few
	// others, as in a real vault.
	fmt.Fprintf(&b, "---\ntitle: Note %d\nstatus: generated\n", i)
	if i > 0 {
		fmt.Fprintf(&b, "links:\n  - to: \"[[%s]]\"\n    role: parent\n", name(i/10))
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# Note %d\n\n", i)

	// One note in a thousand contains a word no other note has. The rest draw on
	// one small vocabulary, where a query matches the whole corpus and measures
	// ranking; a real vault has a long tail of rare terms, which RareTerm queries.
	if i%1000 == 0 {
		b.WriteString(RareTerm + "\n\n")
	}

	// Between ten and four hundred words, in one to four sections: real vaults
	// are not made of one note repeated, and the short ones are as real as the
	// long ones.
	sections := 1 + r.IntN(4)
	for s := range sections {
		fmt.Fprintf(&b, "## Section %d\n\n", s+1)
		for range 10 + r.IntN(90) {
			b.WriteString(vocabulary[r.IntN(len(vocabulary))])
			b.WriteByte(' ')
		}
		// Two or three references into the rest of the vault, spread far enough
		// apart that resolution is not answering from one page of the index.
		for range 2 + r.IntN(2) {
			fmt.Fprintf(&b, "see [[%s]] ", name(r.IntN(total)))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

// name is what a note is called, and therefore what a link by name has to
// resolve.
func name(i int) string { return fmt.Sprintf("note-%06d", i) }
