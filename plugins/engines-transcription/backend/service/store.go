package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

// editPrefix starts the key of a version a person made by editing another.
const editPrefix = "edit-"

// getTranscriptionKey is the key of what the model heard in a recording: the hash of its bytes.
func getTranscriptionKey(fileHash string) string { return fileHash }

// getEditKey is the key of a person's edit of a version. One version has one edit, rewritten as the
// person edits it again.
func getEditKey(input domain.TextVersionID) string {
	sum := sha256.Sum256([]byte(input))
	return editPrefix + hex.EncodeToString(sum[:8])
}

// isEditKey reports whether a key names a person's edit.
func isEditKey(key string) bool { return strings.HasPrefix(key, editPrefix) }

// transcripts are the transcripts kept in the plugin's folder of a vault's store: `<key>.vtt` whole,
// and `<key>.partial.vtt` while a transcription is under way, which holds a note of how far it got
// after every batch of cues it claims.
type transcripts struct {
	store host.Store
}

func getFileName(key string) string    { return key + ".vtt" }
func getPartialName(key string) string { return key + ".partial.vtt" }

// readCues is the cues kept under the key: the whole transcript, else the part a transcription has
// written, else host.ErrNotExist.
func (t transcripts) readCues(ctx context.Context, key string) ([]transcript.Cue, error) {
	raw, err := t.store.Read(ctx, getFileName(key))
	if errors.Is(err, host.ErrNotExist) {
		raw, err = t.store.Read(ctx, getPartialName(key))
	}
	if err != nil {
		return nil, err
	}
	_, cues := transcript.Parse(raw)
	return cues, nil
}

// hasWhole reports whether the whole transcript is kept under the key.
func (t transcripts) hasWhole(ctx context.Context, key string) (bool, error) {
	_, err := t.store.Read(ctx, getFileName(key))
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// readReached is how far a transcription before this one got, in milliseconds, with the bytes past
// the last note cut away: those are a batch that did not land whole.
func (t transcripts) readReached(ctx context.Context, key string) (int, error) {
	raw, err := t.store.Read(ctx, getPartialName(key))
	if errors.Is(err, host.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	ms, end := transcript.ReadReached(raw)
	if end < len(raw) {
		if err := t.store.Write(ctx, getPartialName(key), raw[:end]); err != nil {
			return 0, err
		}
	}
	return ms, nil
}

// appendBatch adds a batch of cues to the partial transcript, and the note that it reached the
// millisecond given after them.
func (t transcripts) appendBatch(ctx context.Context, key string, cues []transcript.Cue, reached int, isFirst bool) error {
	body := transcript.Marshal(cues)
	if !isFirst {
		body = body[len(transcript.Head)+1:]
	}
	return t.store.Append(ctx, getPartialName(key), append(body, transcript.GetReachMarker(reached)...))
}

// finish writes the partial transcript whole under the key and removes the partial one.
func (t transcripts) finish(ctx context.Context, key string) ([]transcript.Cue, error) {
	cues, err := t.readCues(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := t.write(ctx, key, cues); err != nil {
		return nil, err
	}
	return cues, t.store.Delete(ctx, getPartialName(key))
}

// write keeps the cues whole under the key.
func (t transcripts) write(ctx context.Context, key string, cues []transcript.Cue) error {
	return t.store.Write(ctx, getFileName(key), transcript.Marshal(cues))
}

// remove takes away everything kept under the key.
func (t transcripts) remove(ctx context.Context, key string) error {
	return errors.Join(t.store.Delete(ctx, getFileName(key)), t.store.Delete(ctx, getPartialName(key)))
}

// writeParts is the cues as the parts of a timed text, one line to a cue.
func writeParts(cues []transcript.Cue) []domain.TextPart {
	parts := make([]domain.TextPart, len(cues))
	for i, cue := range cues {
		text := cue.Text
		if i < len(cues)-1 {
			text += "\n"
		}
		parts[i] = domain.TextPart{Text: text, Place: domain.TextPlace{
			Start: msToDuration(cue.From), End: msToDuration(cue.To),
		}}
	}
	return parts
}

// hashCues is the hash of the text the cues are as parts.
func hashCues(cues []transcript.Cue) string {
	var b strings.Builder
	for _, part := range writeParts(cues) {
		b.WriteString(part.Text)
	}
	return domain.HashContent([]byte(b.String()))
}
