package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Documents reads a file whose text is laid out on printed pages. Which library
// does it is settled here, with every other choice of adapter.
//
// It holds workers for the life of the process, started when the first document
// is opened.
func (c Config) Documents() port.Documents { return pdf.Documents{} }
