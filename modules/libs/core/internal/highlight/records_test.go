package highlight_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

func TestBoxesComeBackAsTheyWereWritten(t *testing.T) {
	boxes := []highlight.Box{
		testsupport.Box(0, 0, 5, highlight.Rect{MinX: 0, MinY: 0.25, MaxX: 0.5, MaxY: 0.75}),
		testsupport.Box(1, 25, 32, highlight.Rect{MinX: 0.125, MinY: 0.5, MaxX: 1, MaxY: 1}),
		testsupport.Box(17, 4096, 4097, highlight.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4}),
	}
	raw := highlight.Pack(boxes)

	if len(raw) != 3*28 {
		t.Errorf("packed %d bytes for three boxes, want %d", len(raw), 3*28)
	}
	if got := highlight.Unpack(raw); !reflect.DeepEqual(got, boxes) {
		t.Errorf("unpacked %+v, want %+v", got, boxes)
	}
}

func TestATornTailGivesBackTheWholeRecords(t *testing.T) {
	// A run stopped part way through writing leaves a record half written.
	// What was written whole is still a box.
	boxes := []highlight.Box{
		testsupport.Box(0, 0, 5, highlight.Rect{MaxX: 0.5, MaxY: 0.5}),
		testsupport.Box(0, 6, 10, highlight.Rect{MaxX: 0.75, MaxY: 0.5}),
	}
	raw := highlight.Pack(boxes)

	got := highlight.Unpack(raw[:len(raw)-9])
	if len(got) != 1 {
		t.Fatalf("unpacked %d boxes from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], boxes[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], boxes[0])
	}
	if left := highlight.Unpack(raw[:12]); len(left) != 0 {
		t.Errorf("unpacked %d boxes from less than one record", len(left))
	}
}

func BenchmarkPackBox(b *testing.B) {
	for _, count := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("Boxes_%d", count), func(b *testing.B) {
			boxes := make([]highlight.Box, count)
			for i := range count {
				boxes[i] = highlight.Box{
					Page: i / 10,
					ByteSpan: domain.ByteSpan{
						From: i * 20,
						To:   i*20 + 15,
					},
					Rect: highlight.Rect{
						MinX: 0.1,
						MinY: 0.2,
						MaxX: 0.8,
						MaxY: 0.9,
					},
				}
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = highlight.Pack(boxes)
			}
		})
	}
}
