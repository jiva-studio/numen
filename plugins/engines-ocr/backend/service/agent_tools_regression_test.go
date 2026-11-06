package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

func TestRecognitionCreateOfAChangedScanStartsARun(t *testing.T) {
	s := newToolScene(t)
	s.putRecognition(t)
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := append(append([]byte{}, raw...), []byte("\n% a page added later\n")...)
	s.vault.File(scanPath, string(changed))
	s.texts.File(scanPath, changed)

	out := s.recognise(t, "Physics", scanPath)

	if out.State != "started" || out.Version != "" {
		t.Errorf("recognition_create of a changed scan returned %+v, want started", out)
	}
	close(s.recogniser.release)
	s.tools.Close()
}

func TestRecognitionCreateOverAnotherPluginsRecognitionKeyStartsARun(t *testing.T) {
	s := newToolScene(t)
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	other := s.texts.As("md.numen.other")
	if err := host.WriteText(t.Context(), other.GetStore("md.numen.other"), "r.jsonl",
		[]domain.TextSegment{{Text: "other words\n"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := other.PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: recognition.VersionKey, Kind: "recognition", InputHash: domain.HashContent(raw), File: "r.jsonl",
	}); err != nil {
		t.Fatal(err)
	}

	out := s.recognise(t, "Physics", scanPath)

	if out.State != "started" {
		t.Errorf("recognition_create over another plugin's live version returned %+v, want started", out)
	}
	close(s.recogniser.release)
	s.tools.Close()
}

func TestRecognitionCreateInChemistryReadsChemistryVersions(t *testing.T) {
	s := newToolScene(t)
	s.putRecognition(t) // Physics holds a recognition of books/scan.pdf; Chemistry holds none.
	s.chemistry.File(scanPath, "x")

	res := s.recognise(t, "Chemistry", scanPath)

	if res.State == "done" {
		t.Errorf("recognition_create in Chemistry returned %+v from the versions of Physics", res)
	}
	close(s.recogniser.release)
	s.tools.Close()
}
