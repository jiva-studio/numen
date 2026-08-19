package chunk

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
)

// The similarity floor is a number, and a number is chosen by measuring. This
// runs the meaning half over a real vault and prints what each question draws
// out of it: where the answers stand, where the noise stands, and how much of
// each the floor keeps.
//
// It reports; it does not assert. What a question is worth depends on the vault
// it is asked of, and the questions below were written for one.
//
//	NUMEN_FLOOR=1 go test ./internal/adapter/index/chunk/ -run TestTheFloor -v
//
// The index is the installation's own unless NUMEN_FLOOR_INDEX names another,
// and the vault is the first one the index holds unless NUMEN_FLOOR_VAULT names
// it. The embedder is whatever the installation is configured with, so the
// vectors compared are the ones the vault was indexed with.

// floorQuestions are the questions the floor was measured against, each marked
// with whether the vault it was written for holds an answer.
//
// The vault is the Ganguli Mahābhārata beside a few hundred notes on
// mathematics, physics and computation.
var floorQuestions = []struct {
	text string
	held bool
}{
	{"Why did Bhishma stay silent when Draupadi was dragged into the assembly?", true},
	{"the game of dice and the staking of Draupadi", true},
	{"Krishna's counsel to Arjuna before the battle", true},
	{"the death of Karna at the hands of Arjuna", true},
	{"dharma is subtle and hard to know", true},
	{"the house of lac and the escape of the Pandavas", true},
	{"amortised analysis of a dynamic array", true},
	{"a king who does not protect his people", true},
	{"what is a Lagrangian in classical mechanics", true},
	{"chicken", true},
	{"how to configure a Kubernetes ingress controller", false},
	{"sourdough starter hydration ratio", false},
	{"quarterly revenue guidance for a semiconductor supplier", false},
	{"asdkjfh qwpoeiru zxcvbnm", false},
	{"best pizza toppings in Naples", false},
	{"changing the oil filter on a diesel engine", false},
	{"the mating habits of emperor penguins", false},
	{"SELECT * FROM users WHERE deleted_at IS NULL", false},
}

func TestTheFloorSeparatesAnAnswerFromNoise(t *testing.T) {
	if os.Getenv("NUMEN_FLOOR") == "" {
		t.Skip("set NUMEN_FLOOR=1 to measure against a real vault")
	}
	ctx := t.Context()

	cfg, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	embedder, err := openai.New(cfg.Indexing.Embedding.Service)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", floorIndex(t)+"?_pragma=busy_timeout(5000)&_pragma=cache_size(-65536)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := NewQueries(db)
	vault := floorVault(t, ctx, db)

	texts := make([]string, 0, len(floorQuestions))
	for _, q := range floorQuestions {
		texts = append(texts, q.text)
	}
	vectors, err := embedder.Embed(ctx, texts)
	if err != nil {
		t.Fatal(err)
	}

	// The nearest a question the vault holds nothing about got, and the nearest
	// a question it does hold an answer to got. The floor belongs between them.
	var noise, answer float64 = 0, 1

	t.Logf("floor %.3f, %d coarse candidates for each answer, vault %s", Floor, coarseCandidates, vault)
	for i, q := range floorQuestions {
		near, err := queries.coarse(ctx, floorRow(t, ctx, db, vault), vectors[i], 20*coarseCandidates)
		if err != nil {
			t.Fatal(err)
		}
		similarity := floorSimilarities(t, ctx, db, vectors[i], near)

		kept := 0
		for _, s := range similarity {
			if s >= Floor {
				kept++
			}
		}
		if len(similarity) == 0 {
			t.Logf("  %-56.56s  nothing embedded to compare with", q.text)
			continue
		}
		if q.held {
			answer = min(answer, similarity[0])
		} else {
			noise = max(noise, similarity[0])
		}
		t.Logf("  %-56.56s held=%-5v 1=%.4f 8=%.4f 20=%.4f last=%.4f kept=%d/%d",
			q.text, q.held,
			similarity[0], similarity[min(7, len(similarity)-1)], similarity[min(19, len(similarity)-1)],
			similarity[len(similarity)-1], kept, len(similarity))
	}
	t.Logf("the nearest an unheld question got: %.4f", noise)
	t.Logf("the furthest a held question's own answer got: %.4f", answer)
	t.Logf("the floor stands at %.4f", Floor)
}

// floorSimilarities is every candidate's similarity to the query, nearest first.
func floorSimilarities(t *testing.T, ctx context.Context, db *sql.DB, query []float32, candidates []int64) []float64 {
	t.Helper()
	out := make([]float64, 0, len(candidates))
	for _, chunk := range candidates {
		var stored []byte
		err := db.QueryRowContext(ctx, `SELECT v FROM vectors WHERE chunk_id = ?`, chunk).Scan(&stored)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, embedding.Similarity(query, signed(stored)))
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(out)))
	return out
}

func floorIndex(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("NUMEN_FLOOR_INDEX"); path != "" {
		return path
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "numen", "index.db")
}

func floorVault(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	if id := os.Getenv("NUMEN_FLOOR_VAULT"); id != "" {
		return id
	}
	var id string
	if err := db.QueryRowContext(ctx, `SELECT identifier FROM vaults ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func floorRow(t *testing.T, ctx context.Context, db *sql.DB, vaultID string) int64 {
	t.Helper()
	row, err := vaultRow(ctx, db, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	return row
}
