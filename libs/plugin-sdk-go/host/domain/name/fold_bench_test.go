package name

import "testing"

func BenchmarkFoldName(b *testing.B) {
	names := map[string]string{
		"ascii folded": "thermodynamics and information theory",
		"ascii mixed":  "Thermodynamics And Information Theory",
		"non-ascii":    "Śrīmad Bhāgavatam Canto One",
	}
	for label, name := range names {
		b.Run(label, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = FoldName(name)
			}
		})
	}
}
