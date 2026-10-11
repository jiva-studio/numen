package cli

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// DescribeSources is the sentence a scan says about the books it read.
func DescribeSources(r source.ExtractResult) string { return describeSources(r) }
