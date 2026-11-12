package transcribe

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/transcript"
)

// textKind labels the text versions the plugin creates.
const textKind domain.TextKind = "transcript"

func getTranscriptionKey(fileHash string) string { return fileHash }

// transcripts are the transcripts kept in the plugin's folder of a vault's store, in the common
// format of text versions: `<key>.jsonl` whole, and `<key>.partial.jsonl` while a transcription is
// under way. `<key>.reached` records how far the partial transcript got and how many of its bytes
// that is, after every batch of cues it claims.
type transcripts struct {
	store Folder
}

func getFileName(key string) string    { return key + ".jsonl" }
func getPartialName(key string) string { return key + ".partial.jsonl" }
func getReachedName(key string) string { return key + ".reached" }

func isWholeFile(version domain.TextVersion) bool { return version.File == getFileName(version.Key) }

func (t transcripts) hasWhole(ctx context.Context, key string) (bool, error) {
	_, err := t.store.Read(ctx, getFileName(key))
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (t transcripts) readReachedPosition(ctx context.Context, key string) (int, error) {
	raw, err := t.store.Read(ctx, getReachedName(key))
	if errors.Is(err, host.ErrNotExist) {
		return 0, t.store.Delete(ctx, getPartialName(key))
	}
	if err != nil {
		return 0, err
	}
	progress, err := transcript.ParseReached(raw)
	if err != nil {
		return 0, errors.Join(t.store.Delete(ctx, getPartialName(key)), t.store.Delete(ctx, getReachedName(key)))
	}
	partial, err := t.store.Read(ctx, getPartialName(key))
	if errors.Is(err, host.ErrNotExist) {
		return 0, t.store.Delete(ctx, getReachedName(key))
	}
	if err != nil {
		return 0, err
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="writing partial[:progress.Size] when len(partial) equals progress.Size writes identical bytes"
	if int64(len(partial)) > progress.Size {
		if err := t.store.Write(ctx, getPartialName(key), partial[:progress.Size]); err != nil {
			return 0, err
		}
	}
	return progress.MS, nil
}

func (t transcripts) appendBatch(ctx context.Context, key string, cues []transcript.Cue, ms int) error {
	segments, err := transcript.GetSegments(cues)
	// gomutants:disable-next-line BRANCH_IF reason="transcript.GetSegments on finite integer timestamps cannot fail"
	if err != nil {
		return err
	}
	if err := host.AppendText(ctx, t.store, getPartialName(key), segments); err != nil {
		return err
	}
	partial, err := t.store.Read(ctx, getPartialName(key))
	if err != nil {
		return err
	}
	raw, err := transcript.Reached{MS: ms, Size: int64(len(partial))}.Encode()
	// gomutants:disable-next-line BRANCH_IF reason="Reached encodes two integers cannot fail"
	if err != nil {
		return err
	}
	return t.store.Write(ctx, getReachedName(key), raw)
}

func (t transcripts) finish(ctx context.Context, key string) error {
	if err := t.store.Rename(ctx, getPartialName(key), getFileName(key)); err != nil {
		return err
	}
	return t.store.Delete(ctx, getReachedName(key))
}
