package chunk

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/internal/vector"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// scriptedRows is a result set that ends with the error given, or cleanly.
type scriptedRows struct {
	data [][]driver.Value
	fail error
	next int
}

func (r *scriptedRows) Columns() []string { return []string{"id", "embedding"} }
func (r *scriptedRows) Close() error      { return nil }
func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.next < len(r.data) {
		copy(dest, r.data[r.next])
		r.next++
		return nil
	}
	if r.fail != nil {
		return r.fail
	}
	return io.EOF
}

type scriptedConn struct {
	data [][]driver.Value
	fail error
}

func (c *scriptedConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *scriptedConn) Close() error                        { return nil }
func (c *scriptedConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (c *scriptedConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &scriptedRows{data: c.data, fail: c.fail}, nil
}

type scriptedDriver struct{ conn *scriptedConn }

func (d scriptedDriver) Open(string) (driver.Conn, error)             { return d.conn, nil }
func (d scriptedDriver) Connect(context.Context) (driver.Conn, error) { return d.conn, nil }
func (d scriptedDriver) Driver() driver.Driver                        { return d }

// scripted is a Queries whose vector query answers with the rows given.
func scripted(t *testing.T, data [][]driver.Value, fail error) *Queries {
	t.Helper()
	db := sql.OpenDB(scriptedDriver{conn: &scriptedConn{data: data, fail: fail}})
	testsupport.CloseAfter(t, db)
	return NewQueries(db)
}

func signed(values ...int8) []byte {
	out := make([]byte, len(values))
	for i, v := range values {
		out[i] = byte(v)
	}
	return out
}

func idsOf(ids ...int64) []int64 { return ids }

var scriptedQuery = []float32{1, 0, 0, 0}

func TestRerankAnswersNothingForNoCandidates(t *testing.T) {
	q := scripted(t, nil, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, nil, nil, -2)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestRerankAnswersASingleCandidate(t *testing.T) {
	q := scripted(t, [][]driver.Value{{int64(7), signed(127, 0, 0, 0)}}, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(7), nil, -2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, idsOf(7)) {
		t.Errorf("got %v, want [7]", got)
	}
}

func TestRerankOrdersByDescendingSimilarity(t *testing.T) {
	q := scripted(t, [][]driver.Value{
		{int64(10), signed(127, 127, 0, 0)},
		{int64(20), signed(127, 0, 0, 0)},
		{int64(30), signed(0, 127, 0, 0)},
		{int64(40), signed(127, 127, 0, 0)},
		{int64(50), signed(-127, 0, 0, 0)},
	}, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(10, 20, 30, 40, 50), nil, -2)
	if err != nil {
		t.Fatal(err)
	}
	if want := idsOf(20, 10, 40, 30, 50); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRerankKeepsTheArrivalOrderAmongEqualSimilarities(t *testing.T) {
	shapes := [][]int8{{127, 0, 0, 0}, {127, 127, 0, 0}, {127, 127, 127, 0}}
	var data [][]driver.Value
	var candidates, want []int64
	for i := int64(1); i <= 90; i++ {
		data = append(data, []driver.Value{i, signed(shapes[i%3]...)})
		candidates = append(candidates, i)
	}
	// Against {1,0,0,0} the three shapes stand at 1, 0.71 and 0.58, so the
	// answer is each class in turn, and inside a class the rows as they came.
	for _, class := range []int64{0, 1, 2} {
		for i := int64(1); i <= 90; i++ {
			if i%3 == class {
				want = append(want, i)
			}
		}
	}
	q := scripted(t, data, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, candidates, nil, -2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRerankKeepsACandidateAtTheFloor(t *testing.T) {
	stored := signed(127, 127, 0, 0)
	pos := vector.SimilarityBytes(scriptedQuery, stored)
	q := scripted(t, [][]driver.Value{{int64(1), stored}}, nil)

	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(1), nil, pos)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, idsOf(1)) {
		t.Errorf("at the floor: got %v, want [1]", got)
	}
	got, err = q.rerank(t.Context(), "r", scriptedQuery, idsOf(1), nil, math.Nextafter(pos, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("above the floor: got %v, want none", got)
	}
}

func TestRerankRefusesAVectorOfAnotherWidth(t *testing.T) {
	q := scripted(t, [][]driver.Value{{int64(9), signed(1, 2, 3)}}, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(9), nil, -2)
	if err == nil {
		t.Fatalf("got %v and no error", got)
	}
	if got != nil {
		t.Errorf("an error came with %v", got)
	}
}

func TestRerankReportsARowThatCannotBeRead(t *testing.T) {
	q := scripted(t, [][]driver.Value{{"not a number", signed(127, 0, 0, 0)}}, nil)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(1), nil, -2)
	if err == nil {
		t.Fatalf("got %v and no error", got)
	}
	if strings.Contains(err.Error(), "dimensions") {
		t.Errorf("the failure is the width of a vector that was never read: %v", err)
	}
	if got != nil {
		t.Errorf("an error came with %v", got)
	}
}

func TestRerankReportsARowSetThatEndsInFailure(t *testing.T) {
	broken := errors.New("the cursor broke")
	q := scripted(t, [][]driver.Value{{int64(1), signed(127, 0, 0, 0)}}, broken)
	got, err := q.rerank(t.Context(), "r", scriptedQuery, idsOf(1), nil, -2)
	if !errors.Is(err, broken) {
		t.Fatalf("got error %v, want %v", err, broken)
	}
	if got != nil {
		t.Errorf("an error came with %v", got)
	}
}

func TestRerankReportsAFailedQuery(t *testing.T) {
	q, _, query := setupTestDB(t.Context(), t, 3, 8)
	if err := q.db.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := q.rerank(t.Context(), "recipe1", query, idsOf(1, 2, 3), nil, -2)
	if err == nil {
		t.Fatalf("got %v and no error", got)
	}
	if got != nil {
		t.Errorf("an error came with %v", got)
	}
}

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
