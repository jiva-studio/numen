// Package ports declares what the reranking use case asks of the model and of the profile files.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

// Scorer is a loaded cross-encoder.
type Scorer interface {
	// ScorePairs returns one score for each document against query, in the order of the documents.
	ScorePairs(ctx context.Context, query string, documents []string) ([]float32, error)
	Close() error
}

// OpenScorer loads the model of a profile, fetching its files where this machine does not hold them.
type OpenScorer func(ctx context.Context, p profile.Profile) (Scorer, error)

// ProfileSource reads the profiles: the shipped ones and the person's files. A file that cannot be read is
// passed to report and skipped.
type ProfileSource interface {
	Load(report func(file string, err error)) (*profile.Profiles, error)
}
