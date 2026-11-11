package chunk

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestFormatInt64s(t *testing.T) {
	cases := []struct {
		in   []int64
		want string
	}{
		{nil, "[]"},
		{[]int64{5}, "[5]"},
		{[]int64{1, 22, -333}, "[1,22,-333]"},
	}
	for _, c := range cases {
		if got := formatInt64s(c.in); got != c.want {
			t.Errorf("formatInt64s(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatInt64sFillsItsBufferOnce(t *testing.T) {
	for _, nums := range [][]int64{
		{10000000000, 10000000000, 100000000000},
		{10000000000, 20000000000, 30000000000, 40000000000, 50000000000},
	} {
		if n := testing.AllocsPerRun(20, func() { _ = formatInt64s(nums) }); n > 2 {
			t.Errorf("%d numbers took %v allocations, want at most 2", len(nums), n)
		}
	}
}

func TestFormatKinds(t *testing.T) {
	cases := []struct {
		in   []domain.SourceKind
		want string
	}{
		{nil, "[]"},
		{[]domain.SourceKind{"book"}, `["book"]`},
		{[]domain.SourceKind{"book", "note", "recording"}, `["book","note","recording"]`},
	}
	for _, c := range cases {
		if got := formatKinds(c.in); got != c.want {
			t.Errorf("formatKinds(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatKindsFillsItsBufferOnce(t *testing.T) {
	kinds := []domain.SourceKind{"book", "note", "recording"}
	if n := testing.AllocsPerRun(20, func() { _ = formatKinds(kinds) }); n > 2 {
		t.Errorf("took %v allocations, want at most 2", n)
	}
}
