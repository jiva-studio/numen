package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func createLinkNote(count int) []byte {
	var sb strings.Builder
	sb.WriteString("---\ntitle: Linked\nid: 01M02ACGM0FYMSXNDP29C90JNR\nlinks:\n")
	for i := range count {
		fmt.Fprintf(&sb, "  - to: \"[[Note %d]]\"\n    role: related\n    note: why %d\n", i, i)
	}
	sb.WriteString("tags:\n  - a\n---\n\n# Linked\n\nBody text.\n")
	return []byte(sb.String())
}

func BenchmarkLinkBlock(b *testing.B) {
	for _, count := range []int{5, 50} {
		raw := createLinkNote(count)
		target := domain.ParseAddress(fmt.Sprintf("[[Note %d]]", count/2))
		b.Run(fmt.Sprintf("Add/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				doc, _ := Open(raw)
				_ = doc.AddLink(domain.Link{Target: domain.ParseAddress("[[Fresh]]"), Role: domain.LinkRole("parent")})
			}
		})
		b.Run(fmt.Sprintf("Update/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				doc, _ := Open(raw)
				_, _ = doc.UpdateLink(target, domain.Link{Role: domain.LinkRole("parent"), Why: "changed"})
			}
		})
		b.Run(fmt.Sprintf("Remove/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				doc, _ := Open(raw)
				_, _ = doc.RemoveLink(target, "")
			}
		})
		b.Run(fmt.Sprintf("Read/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				doc, _ := Open(raw)
				_, _ = doc.Links()
			}
		})
	}
}
