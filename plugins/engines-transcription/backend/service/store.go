package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

// textKind labels the text versions the plugin creates.
const textKind domain.TextKind = "transcript"

// getTranscriptionKey is the key of what the model heard in a recording: the hash of its bytes.
func getTranscriptionKey(fileHash string) string { return fileHash }

// transcripts are the transcripts kept in the plugin's folder of a vault's store, in the common
// format of text versions: `<key>.jsonl` whole, and `<key>.partial.jsonl` while a transcription is
// under way. `<key>.reached` records how far the partial transcript got and how many of its bytes
// that is, after every batch of cues it claims.
type transcripts struct {
	store host.Store
}

func getFileName(key string) string    { return key + ".jsonl" }
func getPartialName(key string) string { return key + ".partial.jsonl" }
func getReachedName(key string) string { return key + ".reached" }

// reached is how far a partial transcript got: the millisecond of the recording, and the size of
// the partial file at that moment.
type reached struct {
	MS   int   `json:"ms"`
	Size int64 `json:"size"`
}

// hasWhole reports whether the whole transcript is kept under the key.
func (t transcripts) hasWhole(ctx context.Context, key string) (bool, error) {
	_, err := t.store.Read(ctx, getFileName(key))
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// readReached is how far a transcription before this one got, in milliseconds. The bytes of the
// partial transcript past the size recorded with it are a batch that did not land whole, and are
// cut away. A partial transcript with no record of its progress starts again from zero.
func (t transcripts) readReached(ctx context.Context, key string) (int, error) {
	raw, err := t.store.Read(ctx, getReachedName(key))
	if errors.Is(err, host.ErrNotExist) {
		return 0, t.store.Delete(ctx, getPartialName(key))
	}
	if err != nil {
		return 0, err
	}
	var progress reached
	if err := json.Unmarshal(raw, &progress); err != nil {
		return 0, errors.Join(t.store.Delete(ctx, getPartialName(key)), t.store.Delete(ctx, getReachedName(key)))
	}
	partial, err := t.store.Read(ctx, getPartialName(key))
	if errors.Is(err, host.ErrNotExist) {
		return 0, t.store.Delete(ctx, getReachedName(key))
	}
	if err != nil {
		return 0, err
	}
	if int64(len(partial)) > progress.Size {
		if err := t.store.Write(ctx, getPartialName(key), partial[:progress.Size]); err != nil {
			return 0, err
		}
	}
	return progress.MS, nil
}

// appendBatch adds a batch of cues to the partial transcript, and records that it reached the
// millisecond given.
func (t transcripts) appendBatch(ctx context.Context, key string, cues []transcript.Cue, ms int) error {
	if err := host.AppendText(ctx, t.store, getPartialName(key), transcript.GetSegments(cues)); err != nil {
		return err
	}
	partial, err := t.store.Read(ctx, getPartialName(key))
	if err != nil {
		return err
	}
	// A struct of two numbers always marshals.
	raw, _ := json.Marshal(reached{MS: ms, Size: int64(len(partial))})
	return t.store.Write(ctx, getReachedName(key), raw)
}

// finish keeps the partial transcript whole under the key and removes the record of its progress.
func (t transcripts) finish(ctx context.Context, key string) error {
	if err := t.store.Rename(ctx, getPartialName(key), getFileName(key)); err != nil {
		return err
	}
	return t.store.Delete(ctx, getReachedName(key))
}
