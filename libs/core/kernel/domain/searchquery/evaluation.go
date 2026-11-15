package searchquery

import (
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Evaluation is one query over one vault: the query, the scopes of its conditions, and the recipe, floor, limits
// and time zone it is evaluated under.
type Evaluation struct {
	Vault  domain.Vault
	Query  Query
	Scopes []Scope
	Recipe string
	Floor  float64
	Limits Limits
	Zone   *time.Location
}
