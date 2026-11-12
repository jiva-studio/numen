package connection

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrInvalidRows is a value of a setting of rows that is no list of rows.
var ErrInvalidRows = errors.New("the value is no list of rows")

// Row is one entry of a setting of rows: the connection, the model and the reasoning level the person
// chose. The connection is an opaque id; the fields of the connection never reach a plugin.
//
//numen:api
type Row struct {
	// Connection is the id of the connection.
	Connection string `json:"connection"`
	// Model is the name of the model to ask for.
	Model string `json:"model"`
	// Reasoning is the reasoning level; empty asks for the default of the provider.
	Reasoning string `json:"reasoning"`
}

// ReadRows reads the rows of the setting key from the values of Configure, in the order of use. A
// setting that was not delivered has no row.
func ReadRows(values map[string]string, key string) ([]Row, error) {
	text, isDelivered := values[key]
	if !isDelivered {
		return nil, nil
	}
	var rows []Row
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidRows, key, err)
	}
	return rows, nil
}
