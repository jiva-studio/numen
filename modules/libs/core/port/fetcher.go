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

// Metadata is what a site says about an address, before any of it is taken.
type Metadata struct {
	// Title is what the video or the page calls itself.
	Title string
	// Bytes is how large a copy of it would be, and zero where the site does not
	// say.
	Bytes int64
	// Length is how long a video runs, in milliseconds, and zero for an address
	// nothing plays.
	Length int
	// Language is what the site says the video is in, as it names it. A video
	// publishes words in every language somebody has translated it into, and
	// this is the one it was spoken in.
	Language string
	// Captions are the languages a person published words in for it, and
	// Automatic those a machine wrote, both as the site names them. They are
	// two lists because they are worth different amounts: a person's words are
	// what was said, and a machine's are what a machine heard.
	Captions  []string
	Automatic []string
}

// Fetcher is the conversation: what is at this address, and what of it can be
// had. Nothing here reaches the vault — what comes back is bytes, and where
// they are kept is the caller's.
type Fetcher interface {
	// Fetching is what every address this fetcher reaches is fetched by.
	Fetching() FetchModel

	// Metadata is what the site says about the address, taking none of it.
	Metadata(ctx context.Context, at domain.WebAddress) (Metadata, error)

	// Subtitles are the words published with a video in one language, as the site
	// names that language. A video that publishes none in it is
	// ErrNothingFetched.
	//
	// One language is asked for and not a list: a site that publishes a machine
	// translation into every language it knows is asked for one of them, and
	// asking for the lot is hundreds of requests it answers by refusing.
	Subtitles(ctx context.Context, at domain.WebAddress, language string) ([]transcript.Cue, error)

	// Audio is a video's sound, written as the container a transcriber opens.
	// It is what a machine listens to, and is not what a person plays.
	Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error

	// Download is what is at the address as a person plays it, written as it was
	// published. How large it may be is the caller's to hold to.
	Download(ctx context.Context, at domain.WebAddress, into io.Writer) (Download, error)

	// Article is what an address that plays nothing says: the prose a page is
	// written around, without the furniture around it. A page carrying none is
	// ErrNothingFetched.
	Article(ctx context.Context, at domain.WebAddress) (Article, error)
}

// A Download is what a copy came to: what a player is told it is, and the
// extension the bytes are kept under.
type Download struct {
	MediaType string
	Extension string
}

// Article is a page as it reads without the furniture around it.
type Article struct {
	Title string
	Prose string
}
