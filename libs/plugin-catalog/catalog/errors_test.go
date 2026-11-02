package catalog_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
)

func TestSourceErrorsAreFoundInJoinedAndWrappedErrors(t *testing.T) {
	first := &catalog.SourceError{Source: "first", Err: errors.New("offline")}
	second := &catalog.SourceError{Source: "second", Err: errors.New("corrupt")}
	err := fmt.Errorf("listing: %w", errors.Join(catalog.ErrNotFound, errors.Join(first, second)))

	found := catalog.SourceErrors(err)

	if len(found) != 2 || found[0] != first || found[1] != second {
		t.Errorf("found %v, want the two source errors in order", found)
	}
	if got := catalog.SourceErrors(catalog.ErrNotFound); len(got) != 0 {
		t.Errorf("found %v in an error that holds none", got)
	}
	if got := catalog.SourceErrors(nil); len(got) != 0 {
		t.Errorf("found %v in no error", got)
	}
}
