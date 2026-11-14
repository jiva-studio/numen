package textversion

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Texts reads the versions of vault files and their text, and tells which version is live for the
// content a file has.
type Texts struct {
	stores  textport.VersionStores
	files   textport.TextFileStores
	readers vaultport.VaultReaders
	clock   machineport.Clock
	live    live
}

// NewTexts returns the reader of the versions of vault files over the version stores, the file stores,
// the vault readers, the reader of a note's properties and the clock.
func NewTexts(
	stores textport.VersionStores, files textport.TextFileStores, readers vaultport.VaultReaders, notes noteport.NoteReader,
	clock machineport.Clock,
) *Texts {
	return &Texts{stores: stores, files: files, readers: readers, clock: clock, live: live{files: files, notes: notes}}
}

// FindLive lists the versions recorded for the file at the path and finds the live one among them, for
// the content the file has now. A file with no versions has none live and its content is not read.
func (u *Texts) FindLive(ctx context.Context, v domain.Vault, path string) ([]domain.TextVersion, domain.TextVersion, bool, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil || len(versions) == 0 {
		return versions, domain.TextVersion{}, false, err
	}
	contentHash, err := u.GetContentHash(ctx, v, path)
	if err != nil {
		return versions, domain.TextVersion{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	live, hasLive, err := u.live.find(ctx, v, path, versions, contentHash)
	return versions, live, hasLive, err
}

// GetLiveText is the text of the file's live version, or textdomain.ErrNoText.
func (u *Texts) GetLiveText(ctx context.Context, v domain.Vault, path, contentHash string) (domain.VersionText, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	live, ok, err := u.live.find(ctx, v, path, versions, contentHash)
	if err != nil {
		return domain.VersionText{}, err
	}
	if !ok {
		return domain.VersionText{}, fmt.Errorf("%w: %s has no live version", textdomain.ErrNoText, path)
	}
	return u.readText(ctx, v, live)
}

// GetVersionText is the text of one version, live or not, whether or not the
// plugin that made it is enabled. A version the file has not, or whose file
// changed since it was recorded, is textdomain.ErrNoText.
func (u *Texts) GetVersionText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) (domain.VersionText, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	i := indexOf(versions, id)
	if i < 0 {
		return domain.VersionText{}, fmt.Errorf("%w: version %s of %s", textdomain.ErrNoText, id, path)
	}
	return u.readText(ctx, v, versions[i])
}

// pageSize is how many segments one page of a version's text holds.
const pageSize = 1000

// GetTextPage is one page of a version's segments. cursor is the NextCursor of
// the page before, and empty for the first.
func (u *Texts) GetTextPage(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error) {
	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return plugin.TextPage{}, fmt.Errorf("%w: cursor %q", textdomain.ErrNoText, cursor)
		}
		start = n
	}
	version, err := u.GetVersionText(ctx, v, path, id)
	if err != nil {
		return plugin.TextPage{}, err
	}
	segments := version.GetSegments()
	if start > len(segments) {
		return plugin.TextPage{}, fmt.Errorf("%w: cursor %q", textdomain.ErrNoText, cursor)
	}
	end := min(start+pageSize, len(segments))
	page := plugin.TextPage{Segments: segments[start:end]}
	if end < len(segments) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

// GetTextRange returns the segments of a version that cover length bytes of its whole text from
// start, truncated at whole characters. A length of zero reads to the end.
func (u *Texts) GetTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error) {
	if start < 0 || length < 0 {
		return plugin.TextRange{}, fmt.Errorf("invalid range of the text of %s: start %d and length %d must be >= 0", path, start, length)
	}
	version, err := u.GetVersionText(ctx, v, path, id)
	if err != nil {
		return plugin.TextRange{}, err
	}
	size := len(version.Text)
	rangeStart, rangeEnd := min(start, size), size
	if length > 0 && length < size-rangeStart {
		rangeEnd = rangeStart + length
	}
	rangeStart, raw := textdomain.WholeCharacters(rangeStart, []byte(version.Text[rangeStart:rangeEnd]), size)
	return plugin.TextRange{Segments: version.GetSegmentsIn(rangeStart, rangeStart+len(raw)), Start: rangeStart, Size: size}, nil
}

func (u *Texts) readText(ctx context.Context, v domain.Vault, version domain.TextVersion) (domain.VersionText, error) {
	files, err := u.files.Open(v)
	if err != nil {
		return domain.VersionText{}, err
	}
	file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
	if err != nil {
		return domain.VersionText{}, fmt.Errorf("read text %s of %s: %w", version.GetID(), version.Path, err)
	}
	if file.TextHash != version.TextHash {
		return domain.VersionText{}, fmt.Errorf("%w: the file of %s of %s changed", textdomain.ErrNoText, version.GetID(), version.Path)
	}
	return domain.NewVersionText(version, file.Segments), nil
}

// Open is the text of one vault, as the index, the search and the highlighting read it.
func (u *Texts) Open(v domain.Vault) (textdomain.VaultTexts, error) {
	return vaultTexts{texts: u, vault: v}, nil
}

type vaultTexts struct {
	texts *Texts
	vault domain.Vault
}

func (t vaultTexts) FindLiveVersion(ctx context.Context, path, contentHash string) (domain.TextVersionID, error) {
	listing, err := t.texts.ListVersions(ctx, t.vault, path, contentHash)
	if err != nil {
		return "", err
	}
	if listing.Live == "" {
		return "", fmt.Errorf("%w: %s has no live version", textdomain.ErrNoText, path)
	}
	return listing.Live, nil
}

func (t vaultTexts) GetLiveText(ctx context.Context, path, contentHash string) (domain.VersionText, error) {
	return t.texts.GetLiveText(ctx, t.vault, path, contentHash)
}

func (t vaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	return t.texts.GetVersionText(ctx, t.vault, path, id)
}

func (t vaultTexts) IsTextSource(ctx context.Context, path string) (bool, error) {
	return t.texts.IsTextSource(ctx, t.vault, path)
}

func (t vaultTexts) GetContentHash(ctx context.Context, path string) (string, error) {
	return t.texts.GetContentHash(ctx, t.vault, path)
}
