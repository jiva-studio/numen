package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// TextExtractor takes the text out of a file whose text is laid out on printed
// pages. Which library does it is settled here, with every other choice of
// adapter.
func (c Config) TextExtractor() port.TextExtractor { return c.getDocuments() }

// PageRenderer draws such a document's pages. The library behind it holds
// workers for the life of the process, started when the first document is
// opened.
func (c Config) PageRenderer() port.PageRenderer { return c.getDocuments() }

// getDocuments is the document library, its worker pool sized by the compute plan.
// A container without a plan leaves the pool at one worker per CPU.
func (c Config) getDocuments() pdf.Documents {
	pdf.SetWorkers(c.ResolvedCompute.Workers)
	return pdf.Documents{}
}
