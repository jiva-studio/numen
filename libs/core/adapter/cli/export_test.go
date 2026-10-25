package cli

import "github.com/jiva-studio/numen/libs/core/usecase/source"

// DescribeSources is the sentence a scan says about the books it read.
func DescribeSources(r source.ExtractResult) string { return describeSources(r) }

// Excerpt is the line one passage takes in the terminal's output.
func Excerpt(text string) string { return getExcerpt(text) }
