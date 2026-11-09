package domain_test

import (
	"bytes"
	"errors"
	"testing"
	"testing/iotest"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestHashReaderEqualsHashContentOfTheSameBytes(t *testing.T) {
	for name, raw := range map[string][]byte{
		"empty":         {},
		"short":         []byte("%PDF atlas of rivers"),
		"several reads": bytes.Repeat([]byte("delta of the river "), 10_000),
	} {
		got, err := domain.HashReader(iotest.HalfReader(bytes.NewReader(raw)))
		if err != nil || got != domain.HashContent(raw) {
			t.Errorf("%s: HashReader gives %s (%v), want %s", name, got, err, domain.HashContent(raw))
		}
	}
}

func TestHashReaderReturnsTheFailureOfTheStream(t *testing.T) {
	failure := errors.New("the disk went away")
	if _, err := domain.HashReader(iotest.ErrReader(failure)); !errors.Is(err, failure) {
		t.Fatalf("got %v, want the stream's failure", err)
	}
}
