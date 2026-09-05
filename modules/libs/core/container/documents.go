package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// TextExtractor takes the text out of a file whose text is laid out on printed
// pages. Which library does it is settled here, with every other choice of
// adapter.
func (c Config) TextExtractor() port.TextExtractor { return pdf.Documents{} }

// PageRenderer draws such a document's pages. The library behind it holds
// workers for the life of the process, started when the first document is
// opened.
func (c Config) PageRenderer() port.PageRenderer { return pdf.Documents{} }
