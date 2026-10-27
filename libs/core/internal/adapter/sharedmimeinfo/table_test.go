package sharedmimeinfo_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestGetContentTypeFindsTheTypeOfAName(t *testing.T) {
	cases := map[string]domain.ContentType{
		"papers/Lecture.pdf":     "application/pdf",
		"talk.mp3":               "audio/mpeg",
		"talk.flac":              "audio/flac",
		"class.mp4":              "video/mp4",
		"book.epub":              "application/epub+zip",
		"notes/plan.md":          "text/markdown",
		"site.url":               "application/x-mswinurl",
		"Makefile":               "text/x-makefile",
		"papers/lecture.pdf.gz":  "application/x-gzpdf",
		"archive.tar.gz":         "application/x-compressed-tar",
		"libnumen.so.1":          "application/x-sharedlib",
		"no-extension":           domain.UnknownContentType,
		"papers/lecture.unknown": domain.UnknownContentType,
	}
	table := sharedmimeinfo.Open()
	for name, want := range cases {
		if got := table.GetContentType(name); got != want {
			t.Errorf("GetContentType(%q) = %q; want %q", name, got, want)
		}
	}
}

func TestGetContentTypeIgnoresTheCaseOfAnExtension(t *testing.T) {
	table := sharedmimeinfo.Open()
	for _, name := range []string{"LECTURE.PDF", "Lecture.Pdf", "lecture.pdf"} {
		if got := table.GetContentType(name); got != "application/pdf" {
			t.Errorf("GetContentType(%q) = %q; want application/pdf", name, got)
		}
	}
}

func TestGetContentTypeKeepsTheCaseOfACaseSensitivePattern(t *testing.T) {
	table := sharedmimeinfo.Open()
	if got := table.GetContentType("main.c"); got != "text/x-csrc" {
		t.Errorf("GetContentType(main.c) = %q; want text/x-csrc", got)
	}
	if got := table.GetContentType("main.C"); got != "text/x-c++src" {
		t.Errorf("GetContentType(main.C) = %q; want text/x-c++src", got)
	}
}

func TestGetContentTypePrefersTheHeavierPattern(t *testing.T) {
	if got := sharedmimeinfo.Open().GetContentType("readme.md"); got != "text/markdown" {
		t.Errorf("GetContentType(readme.md) = %q; want text/markdown over a ROM image", got)
	}
}

func TestListExtensionsListsTheExtensionsOfAMask(t *testing.T) {
	got := sharedmimeinfo.Open().ListExtensions("audio/*")
	for _, want := range []string{"mp3", "wav", "flac", "m4a", "ogg"} {
		if !slices.Contains(got, want) {
			t.Errorf("ListExtensions(audio/*) = %v; want %s among them", got, want)
		}
	}
	if slices.Contains(got, "mp4") || slices.Contains(got, "pdf") {
		t.Errorf("ListExtensions(audio/*) = %v; want no video or document", got)
	}
}

func TestListExtensionsExcludesAnExtensionOfAHeavierType(t *testing.T) {
	if got := sharedmimeinfo.Open().ListExtensions("application/x-genesis-rom"); slices.Contains(got, "md") {
		t.Errorf("ListExtensions = %v; want md left to text/markdown", got)
	}
}

func TestListExtensionsListsOneExtensionOfAType(t *testing.T) {
	if got := sharedmimeinfo.Open().ListExtensions("application/pdf"); !slices.Equal(got, []string{"pdf"}) {
		t.Errorf("ListExtensions(application/pdf) = %v; want [pdf]", got)
	}
}
