package port

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
)

// ErrNothingDownloaded is an address that answered, and carries nothing of what
// was asked for: a video nobody published words for, a page with no prose in
// it. It is an answer and not a failure.
var ErrNothingDownloaded = errors.New("nothing of the sort is published there")

// DownloadModel names what downloaded, and is kept beside what it brought back.
// What a site publishes and what a tool takes out of it both change, and a text
// kept beyond the run that made it is claimed again by what made it.
type DownloadModel struct {
	// Tool is what was run, and Version what it answered when asked which it
	// is.
	Tool    string
	Version string
	// Producer is what this provider's texts are kept under, which is asked
	// before anything is downloaded: an address that publishes nothing still
	// has that answer written down under a name.
	Producer string
}

// Recipe is everything about this download that decides what a text is, as one
// value.
func (f DownloadModel) Recipe() string { return f.Tool + "|" + f.Version }

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

// Downloader is the conversation: what is at this address, and what of it can
// be had. Nothing here reaches the vault — what comes back is bytes, and where
// they are kept is the caller's.
type Downloader interface {
	// Downloading is what this address is downloaded by. A video and a page are
	// reached by different tools, and what is kept names the one that brought
	// it.
	Downloading(at domain.URL) DownloadModel

	// Metadata is what the site says about the address, taking none of it.
	Metadata(ctx context.Context, at domain.URL) (Metadata, error)

	// Text is what the address publishes as words. Which words those are is the
	// provider's to say and the caller's to keep: a site that publishes them
	// against a clock answers with cues, one that publishes prose answers with
	// prose, and the provider names what made them. An address publishing none
	// of what was asked for is ErrNothingDownloaded.
	Text(ctx context.Context, at domain.URL, want PreferredCaptions) (Text, error)

	// Download is what is at the address as a person plays it, written as it was
	// published. How large it may be is the caller's to hold to.
	Download(ctx context.Context, at domain.URL, into io.Writer) (Copy, error)
}

// PreferredCaptions is which of the words a site published the caller will
// take: the languages, best first, and whether words a machine wrote count
// where a person published none.
//
// A site that publishes a machine translation into every language it knows is
// asked for one of them, and asking for the lot is hundreds of requests it
// answers by refusing. Which one that is, the provider picks by these. A site
// publishing prose has one text and ignores them.
type PreferredCaptions struct {
	Languages []string
	Automatic bool
}

// Text is what an address publishes as words.
//
// Cues and Prose are the two shapes it comes in and one of them is set: words
// against the clock they were said on, or prose nothing timed.
type Text struct {
	// Producer names what made these words, and is kept beside them: what a
	// site publishes and what a tool takes out of it both change, and a text
	// kept beyond the run that made it is claimed again by what made it.
	Producer string

	Cues  []transcript.Cue
	Prose string

	// Title is what it calls itself, and Length how long it runs in
	// milliseconds where anything runs.
	Title  string
	Length int
}

// A Copy is what a downloaded copy came to: what a player is told it is, and
// the extension the bytes are kept under.
type Copy struct {
	MediaType string
	Extension string
}
