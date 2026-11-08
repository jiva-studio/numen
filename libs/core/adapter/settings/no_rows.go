package settings

import "github.com/jiva-studio/numen/libs/core/port/connectionport"

// NoRows is the rows of a settings file that holds no setting of rows: no plugin names a connection.
type NoRows struct{}

// ListRows returns no row.
func (NoRows) ListRows(string, string) ([]connectionport.Row, error) { return nil, nil }
