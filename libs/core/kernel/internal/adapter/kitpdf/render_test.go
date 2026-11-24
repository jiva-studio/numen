package kitpdf_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/kitpdf"
)

const fingerprint = "1A2B 3C4D 5E6F 7A8B 9C0D 1E2F 3A4B 5C6D"
const chainMark = "7-3FA912C077B1"

var (
	pageObject = regexp.MustCompile(`/Type /Page\b[^s]`)
	stream     = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	shown      = regexp.MustCompile(`\(((?:[^()\\]|\\.)*)\) Tj`)
	lowerWord  = regexp.MustCompile(`^[a-z]+$`)
)

func getSampleKit() identitydomain.Kit {
	sheets := make([]identitydomain.Sheet, 3)
	for i := range sheets {
		words := make([]string, 33)
		for w := range words {
			words[w] = string(rune('a'+i)) + strings.Repeat("x", w%7+1) + string(rune('a'+w%26))
		}
		sheets[i] = identitydomain.Sheet{Number: i + 1, Words: words}
	}
	return identitydomain.Kit{
		ID:                 "C0DE",
		AccountFingerprint: fingerprint,
		ChainMark:          chainMark,
		CreatedAt:          time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
		Sheets:             sheets,
	}
}

func readPages(t *testing.T, content []byte) [][]string {
	t.Helper()
	var pages [][]string
	for _, found := range stream.FindAllSubmatch(content, -1) {
		reader, err := zlib.NewReader(bytes.NewReader(found[1]))
		if err != nil {
			continue
		}
		text, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("inflate a stream: %v", err)
		}
		var strs []string
		for _, one := range shown.FindAllSubmatch(text, -1) {
			strs = append(strs, string(one[1]))
		}
		if len(strs) > 0 {
			pages = append(pages, strs)
		}
	}
	return pages
}

func TestRender_WritesAPDFOfOnePageForEachSheet(t *testing.T) {
	t.Parallel()
	content, err := kitpdf.Render(getSampleKit())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("the document starts with %q", content[:min(8, len(content))])
	}
	if got := len(pageObject.FindAll(content, -1)); got != 3 {
		t.Errorf("the document has %d pages, want 3", got)
	}
	if pages := readPages(t, content); len(pages) != 3 {
		t.Errorf("%d pages show text, want 3", len(pages))
	}
}

func TestRender_ShowsTheFingerprintTheRuleAndTheWordsOfEachSheetInOrder(t *testing.T) {
	t.Parallel()
	kit := getSampleKit()
	content, err := kitpdf.Render(kit)
	if err != nil {
		t.Fatal(err)
	}
	for i, page := range readPages(t, content) {
		for _, want := range []string{
			"Numen recovery kit",
			fingerprint,
			"Any 2 of 3 sheets restore your account. One sheet alone is useless.",
			"Keep each sheet in a different place. Never store the sheets together.",
			"23 November 2026",
			"Kept at:",
			"Scan this sheet to restore",
			chainMark,
		} {
			if !slices.Contains(page, want) {
				t.Errorf("page %d does not show %q", i+1, want)
			}
		}
		if !slices.ContainsFunc(page, func(s string) bool { return strings.Contains(s, "Sheet "+string(rune('1'+i))+" of 3") }) {
			t.Errorf("page %d does not name its sheet", i+1)
		}
		var words []string
		for _, s := range page {
			if lowerWord.MatchString(s) {
				words = append(words, s)
			}
		}
		if !slices.Equal(words, kit.Sheets[i].Words) {
			t.Errorf("page %d shows the words %v, want %v", i+1, words, kit.Sheets[i].Words)
		}
	}
}

func TestRender_ShowsNoWordOfAnotherSheet(t *testing.T) {
	t.Parallel()
	kit := getSampleKit()
	content, err := kitpdf.Render(kit)
	if err != nil {
		t.Fatal(err)
	}
	pages := readPages(t, content)
	for i, page := range pages {
		for j, sheet := range kit.Sheets {
			if i == j {
				continue
			}
			for _, word := range sheet.Words {
				if slices.Contains(page, word) {
					t.Errorf("page %d shows %q of sheet %d", i+1, word, j+1)
				}
			}
		}
	}
}

type dialog struct {
	path  string
	fails error
	asked []string
}

func (d *dialog) ChooseSaveFile(_ context.Context, title, name string, accept []string) (string, bool, error) {
	d.asked = append(d.asked, title, name, strings.Join(accept, ","))
	return d.path, d.path != "", d.fails
}

func TestSaver_WritesThePDFOnlyThePersonCanReadToTheChosenFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kit.pdf")
	asked := &dialog{path: path}
	isSaved, err := kitpdf.NewSaver(asked).SaveKit(t.Context(), getSampleKit())
	if err != nil || !isSaved {
		t.Fatalf("SaveKit() = %v, %v", isSaved, err)
	}
	if !slices.Equal(asked.asked, []string{"Save the recovery kit", "Numen recovery kit.pdf", ".pdf"}) {
		t.Errorf("the dialog was asked %q", asked.asked)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Error("the file is not a PDF")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 && !isWindows() {
		t.Errorf("the file has the mode %v, want only the owner", mode)
	}
}

func TestSaver_WritesNothingWhenNoFileIsChosen(t *testing.T) {
	t.Parallel()
	isSaved, err := kitpdf.NewSaver(&dialog{}).SaveKit(t.Context(), getSampleKit())
	if err != nil || isSaved {
		t.Errorf("SaveKit() = %v, %v; want false and no error", isSaved, err)
	}
}

func TestSaver_ReportsADialogThatFails(t *testing.T) {
	t.Parallel()
	_, err := kitpdf.NewSaver(&dialog{fails: machine.ErrChoosing}).SaveKit(t.Context(), getSampleKit())
	if !errors.Is(err, machine.ErrChoosing) {
		t.Errorf("error = %v, want the error of the dialog", err)
	}
	if _, err := (kitpdf.Unavailable{}).SaveKit(t.Context(), getSampleKit()); !errors.Is(err, machine.ErrNoFolderDialog) {
		t.Errorf("Unavailable: error = %v, want ErrNoFolderDialog", err)
	}
}

func isWindows() bool { return filepath.Separator == '\\' }
