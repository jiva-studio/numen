package cli

import (
	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
)

// DescribeSources is the sentence a scan reports about the books it read.
func DescribeSources(r indexdomain.IndexResult) string { return describeSources(r) }

// Excerpt is the line one passage takes in the terminal's output.
func Excerpt(text string) string { return getExcerpt(text) }
