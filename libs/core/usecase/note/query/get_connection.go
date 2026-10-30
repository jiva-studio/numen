package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// GetConnection reads the relationship between two notes as a whole: the record
// each note writes about the other.
type GetConnection struct {
	links noteport.LinkQueries
}

// NewGetConnection builds the use case over the link queries of the index.
func NewGetConnection(links noteport.LinkQueries) GetConnection {
	return GetConnection{links: links}
}

// Execute returns the connection between a and b of one type. The empty type
// selects the records that carry none. A note that writes no record about the
// other has a nil side. Only roles a connection can carry take part: a body
// reference is not a record of the connection.
func (u GetConnection) Execute(
	ctx context.Context, v domain.Vault, a, b, kind string,
) (domain.Connection, error) {
	ab, err := u.getSide(ctx, v, a, b, kind)
	if err != nil {
		return domain.Connection{}, err
	}
	ba, err := u.getSide(ctx, v, b, a, kind)
	if err != nil {
		return domain.Connection{}, err
	}
	return domain.Connection{A: a, B: b, AB: ab, BA: ba}, nil
}

// getSide returns the first record written in from that resolves to to.
func (u GetConnection) getSide(
	ctx context.Context, v domain.Vault, from, to, kind string,
) (*domain.Side, error) {
	links, err := u.links.GetLinks(ctx, v.ID, from)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.ToVault != v.ID || l.To != to || l.Type != kind || !domain.IsConnectionRole(l.Role) {
			continue
		}
		return &domain.Side{Role: l.Role, Type: l.Type, Why: l.Why, Label: l.Label, Marker: l.Marker}, nil
	}
	return nil, nil
}
