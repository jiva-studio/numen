package note

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestAMarkedNameIsReadBackAsTheNameAndTheRunsInIt(t *testing.T) {
	for _, c := range []struct {
		marked string
		name   string
		at     []domain.UnitSpan
	}{
		{"Entropy", "Entropy", nil},
		{"\x02Entropy\x03", "Entropy", []domain.UnitSpan{{From: 0, To: 7}}},
		{"The \x02Carnot\x03 cycle", "The Carnot cycle", []domain.UnitSpan{{From: 4, To: 10}}},
		{
			"\x02Heat\x03 and \x02work\x03",
			"Heat and work",
			[]domain.UnitSpan{{From: 0, To: 4}, {From: 9, To: 13}},
		},
		// A run with nothing in it says nothing, and is not one.
		{"\x02\x03Entropy", "Entropy", nil},
		// A character outside the basic plane is two code units to a client,
		// and a run after one begins that much further along.
		{"👋 \x02Entropy\x03", "👋 Entropy", []domain.UnitSpan{{From: 3, To: 10}}},
		{"\x02👋\x03 Entropy", "👋 Entropy", []domain.UnitSpan{{From: 0, To: 2}}},
	} {
		name, at := split(c.marked)
		if name != c.name {
			t.Errorf("split(%q) read %q, want %q", c.marked, name, c.name)
		}
		if len(at) != len(c.at) {
			t.Errorf("split(%q) marked %+v, want %+v", c.marked, at, c.at)
			continue
		}
		for i := range at {
			if at[i] != c.at[i] {
				t.Errorf("split(%q) marked %+v, want %+v", c.marked, at, c.at)
				break
			}
		}
	}
}

func BenchmarkSplit(b *testing.B) {
	for _, c := range []struct{ name, marked string }{
		{"plain", "Entropy and the second law of thermodynamics"},
		{"one-run", "The \x02Carnot\x03 cycle and heat engines"},
		{"many-runs", "\x02Heat\x03 \x02and\x03 \x02work\x03 in \x02the\x03 \x02Carnot\x03 \x02cycle\x03 of \x02a\x03 \x02heat\x03 \x02engine\x03"},
		{"unicode", "👋 \x02Entropy\x03 😀 \x02энтропия\x03 \x02熵\x03 🔥\x02heat\x03"},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				split(c.marked)
			}
		})
	}
}

func splitReference(marked string) (string, []domain.UnitSpan) {
	var name strings.Builder
	var at []domain.UnitSpan

	units, from := 0, -1
	for _, r := range marked {
		switch r {
		case '\x02':
			from = units
		case '\x03':
			if from >= 0 && units > from {
				at = append(at, domain.UnitSpan{From: from, To: units})
			}
			from = -1
		default:
			name.WriteRune(r)
			units++
			if r > 0xffff {
				units++
			}
		}
	}
	return name.String(), at
}

func TestSplitReadsEveryMarkedNameAsTheReferenceDoes(t *testing.T) {
	alphabet := []rune{'a', 'Z', ' ', 'é', 'э', '熵', '👋', '😀', '\U0010FFFF', '\x02', '\x03', '\x02', '\x03'}
	rng := rand.New(rand.NewPCG(367, 1))
	for i := 0; i < 20000; i++ {
		runes := make([]rune, rng.IntN(40))
		for j := range runes {
			runes[j] = alphabet[rng.IntN(len(alphabet))]
		}
		marked := string(runes)

		wantName, wantAt := splitReference(marked)
		gotName, gotAt := split(marked)
		if gotName != wantName {
			t.Fatalf("split(%q) read %q, want %q", marked, gotName, wantName)
		}
		if len(gotAt) != len(wantAt) {
			t.Fatalf("split(%q) marked %+v, want %+v", marked, gotAt, wantAt)
		}
		for k := range gotAt {
			if gotAt[k] != wantAt[k] {
				t.Fatalf("split(%q) marked %+v, want %+v", marked, gotAt, wantAt)
			}
		}
	}
}
