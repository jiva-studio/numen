package vault

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestAnErrorTheHostCannotNameIsInternal(t *testing.T) {
	_, err := answer(t.Context(), errors.New("the disk is gone"))

	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("the failure was %v", err)
	}
}

func TestAFingerprintCarriesWhetherTheFileIsANote(t *testing.T) {
	for _, kind := range []domain.SourceKind{domain.KindNote, domain.KindFile} {
		got := readFileStat(writeFileStat(domain.FileStat{Path: "B.sample", Kind: kind, Size: 3, ModTime: time.Unix(5, 6)}))

		if got.Kind != kind || got.Path != "B.sample" || !got.ModTime.Equal(time.Unix(5, 6)) {
			t.Errorf("got %+v, want the kind %q", got, kind)
		}
	}
	if got := readFileStat(&v1.HostFileStat{}); got.Kind != "" {
		t.Errorf("got %+v, want a fingerprint naming no file to have no kind", got)
	}
}
