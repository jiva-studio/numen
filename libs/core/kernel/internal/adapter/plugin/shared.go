package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/contenttypes"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/errsink"
)

// NewErrorSink creates the sink the roles of a host report the errors they meet outside a call to.
func NewErrorSink() *errsink.Sink { return &errsink.Sink{} }

// NewContentTypeLookup creates the lookup that holds the table of content types, for the registry that lists the
// readers and the content types that gives a file its type.
func NewContentTypeLookup() *contenttypes.Lookup { return &contenttypes.Lookup{} }
