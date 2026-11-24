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
	"strconv"
	"strings"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/kitpdf"
)

var (
	pageObject = regexp.MustCompile(`/Type /Page\b[^s]`)
	stream     = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	shown      = regexp.MustCompile(`BT ([\d.-]+) ([\d.-]+) Td \(((?:[^()\\]|\\.)*)\) Tj`)
)

// shownText is one string the page shows and the point, in points, where it starts.
type shownText struct {
	x, y float64
	text string
}

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
		ID:        "C0DE",
		CreatedAt: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
		Sheets:    sheets,
	}
}

func readTexts(t *testing.T, content []byte) [][]shownText {
	t.Helper()
	var pages [][]shownText
	for _, found := range stream.FindAllSubmatch(content, -1) {
		reader, err := zlib.NewReader(bytes.NewReader(found[1]))
		if err != nil {
			continue
		}
		text, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("inflate a stream: %v", err)
		}
		var texts []shownText
		for _, one := range shown.FindAllSubmatch(text, -1) {
			x, errX := strconv.ParseFloat(string(one[1]), 64)
			y, errY := strconv.ParseFloat(string(one[2]), 64)
			if errX != nil || errY != nil {
				t.Fatalf("read the point of %q", one[3])
			}
			texts = append(texts, shownText{x: x, y: y, text: string(one[3])})
		}
		if len(texts) > 0 {
			pages = append(pages, texts)
		}
	}
	return pages
}

func readPages(t *testing.T, content []byte) [][]string {
	t.Helper()
	var pages [][]string
	for _, texts := range readTexts(t, content) {
		strs := make([]string, len(texts))
		for i, one := range texts {
			strs[i] = one.text
		}
		pages = append(pages, strs)
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

func TestRender_ShowsTheTextsOfTheDesign(t *testing.T) {
	t.Parallel()
	kit := getSampleKit()
	content, err := kitpdf.Render(kit)
	if err != nil {
		t.Fatal(err)
	}
	for i, page := range readPages(t, content) {
		number := string(rune('1' + i))
		joined := strings.Join(strings.Fields(strings.Join(page, " ")), " ")
		for _, want := range []string{
			"Numen", "Recovery kit", "Sheet " + number + " of 3",
			"Restore by scanning", "Restore by typing",
			"Settings > Identity", "Restore > Recovery kit: scan", "Restore > Recovery kit: type",
			"QR CODE OF SHEET " + number, "THE 33 WORDS OF SHEET " + number,
			"numen.md", "Do not photograph or share. Anyone with two sheets can restore your identity.",
		} {
			if !slices.Contains(page, want) {
				t.Errorf("page %d does not show %q", i+1, want)
			}
		}
		for _, want := range []string{
			"This sheet helps you get your identity back if you lose your computer.",
			"Any two of the three sheets are enough.",
			"One sheet alone cannot be used by anyone. Keep each sheet in a different place.",
			"Scan the QR code of this sheet, then of another.",
			"Type the 33 words of this sheet, then of another.",
			"It holds the same 33 words as the list below. Both ways give the same result.",
		} {
			if !strings.Contains(joined, want) {
				t.Errorf("page %d does not show %q", i+1, want)
			}
		}
		if strings.Contains(strings.ToLower(joined), "account") {
			t.Errorf("page %d says account, the product says identity", i+1)
		}
	}
}

func TestRender_ShowsTheWordsOfTheSheetInThreeColumnsOfEleven(t *testing.T) {
	t.Parallel()
	kit := getSampleKit()
	content, err := kitpdf.Render(kit)
	if err != nil {
		t.Fatal(err)
	}
	for i, page := range readTexts(t, content) {
		words := kit.Sheets[i].Words
		var columns []float64
		byColumn := map[float64][]string{}
		for _, one := range page {
			if !slices.Contains(words, one.text) {
				continue
			}
			if _, ok := byColumn[one.x]; !ok {
				columns = append(columns, one.x)
			}
			byColumn[one.x] = append(byColumn[one.x], one.text)
		}
		if len(columns) != 3 {
			t.Fatalf("page %d shows the words at %d left edges, want 3", i+1, len(columns))
		}
		for c, x := range columns {
			if want := words[c*11 : (c+1)*11]; !slices.Equal(byColumn[x], want) {
				t.Errorf("page %d column %d shows %v, want %v", i+1, c+1, byColumn[x], want)
			}
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
