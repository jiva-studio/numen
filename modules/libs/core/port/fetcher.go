package port

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ErrNothingFetched is an address that answered, and carries nothing of what
// was asked for: a video nobody published words for, a page with no prose in
// it. It is an answer and not a failure.
var ErrNothingFetched = errors.New("nothing of the sort is published there")

// FetchModel names what fetched, and is kept beside what it brought back. What
// a site publishes and what a tool takes out of it both change, and a text kept
// beyond the run that made it is claimed again by what made it.
type FetchModel struct {
	// Tool is what was run, and Version what it answered when asked which it
	// is.
	Tool    string
	Version string
}

// Recipe is everything about this fetch that decides what a text is, as one
// value.
func (f FetchModel) Recipe() string { return f.Tool + "|" + f.Version }

// Found is what stands at an address, before any of it is taken.
type Found struct {
	// Title is what the video or the page calls itself.
	Title string
	// Length is how long a video runs, in milliseconds, and zero for an address
	// nothing plays.
	Length int
	// Captions are the languages words are published in for it, as the site
	// names them. A language a machine wrote is named here like any other.
	Captions []string
}

// Fetcher is the conversation: what is at this address, and what of it can be
// had. Nothing here reaches the vault — what comes back is bytes, and where
// they are kept is the caller's.
type Fetcher interface {
	// Fetching is what every address this fetcher reaches is fetched by.
	Fetching() FetchModel

	// Look is what stands at the address, taking none of it.
	Look(ctx context.Context, at domain.WebAddress) (Found, error)

	// Words are the words published with a video, in the first of those
	// languages the site has. A video nobody published any for is
	// ErrNothingFetched.
	Words(ctx context.Context, at domain.WebAddress, languages []string) ([]transcript.Cue, error)

	// Sound is a video's sound, written as the container a transcriber opens.
	// It is what a machine listens to, and is not what a person plays.
	Sound(ctx context.Context, at domain.WebAddress, into io.Writer) error

	// Copy is what is at the address as a person plays it, written as it was
	// published. How large it may be is the caller's to hold to.
	Copy(ctx context.Context, at domain.WebAddress, into io.Writer) (CopyResult, error)

	// Prose is what an address that plays nothing says: the article a page is
	// written around, without the furniture around it. A page carrying none is
	// ErrNothingFetched.
	Prose(ctx context.Context, at domain.WebAddress) (Article, error)
}

// CopyResult is what a copy came to: what a player is told it is, and the
// extension the bytes are kept under.
type CopyResult struct {
	MediaType string
	Extension string
}

// Article is a page as it reads without the furniture around it.
type Article struct {
	Title string
	Prose string
}
