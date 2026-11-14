package openaiapi

import (
	"fmt"
	"os"
)

// ReportListError writes a failure to refresh or keep a list of models to stderr; the list kept before is
// still answered.
func ReportListError(err error) {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="a diagnostic line on stderr; the list kept before is still answered"
	fmt.Fprintf(os.Stderr, "list models: %v\n", err)
}
