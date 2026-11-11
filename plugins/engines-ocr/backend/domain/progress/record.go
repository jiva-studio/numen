// Package progress holds the record of a recognition that has not ended and the text version made
// from it.
package progress

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// TextKind labels the text versions the plugin creates.
const TextKind domain.TextKind = "recognition"

// Record is the progress of a recognition that has not ended, kept in the plugin's folder of the
// vault's store between batches: the content it reads, what reads it, and the text of each page read
// so far. A recognition stopped part way takes it up again where it stopped. The record is deleted
// once the version is put, so the folder holds only the version file of a recognized path.
type Record struct {
	Path      string   `json:"path"`
	InputHash string   `json:"inputHash"`
	Recipe    string   `json:"recipe"`
	PageCount int      `json:"pageCount"`
	Pages     [][]Part `json:"pages"`
}

// Part is one run of a page's text and the rectangles it was read from, in fractions of the page.
type Part struct {
	Text    string       `json:"text"`
	Rects   [][4]float32 `json:"rects,omitempty"`
	Section string       `json:"section,omitempty"`
}

// GetRecordName is the name of the file that keeps the record of the file at path.
func GetRecordName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".progress.json"
}

// GetVersionName is the name of the file that keeps the text version of the file at path.
func GetVersionName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".jsonl"
}

// ParseRecord reads a record from the bytes Encode wrote.
func ParseRecord(raw []byte) (*Record, error) {
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Encode returns the bytes ParseRecord reads.
func (r *Record) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// IsFor reports whether the record was made from the same content with the same recipe.
func (r *Record) IsFor(inputHash, recipe string) bool {
	return r != nil && r.InputHash == inputHash && r.Recipe == recipe
}

// GetTextSegments returns the text segments of the page at index, counted from zero.
func (r *Record) GetTextSegments(index int) ([]domain.TextSegment, error) {
	out := make([]domain.TextSegment, 0, len(r.Pages[index]))
	for _, one := range r.Pages[index] {
		selector, err := pageSelector{Page: index + 1, Boxes: one.Rects}.encode()
		if err != nil {
			return nil, err
		}
		out = append(out, domain.TextSegment{Text: one.Text, Heading: one.Section, Selector: selector})
	}
	return out, nil
}

// GetText returns the text of all the pages read so far.
func (r *Record) GetText() (domain.VersionText, error) {
	var segments []domain.TextSegment
	for index := range r.Pages {
		page, err := r.GetTextSegments(index)
		if err != nil {
			return domain.VersionText{}, err
		}
		segments = append(segments, page...)
	}
	return domain.NewVersionText(domain.TextVersion{
		Path: "", PluginID: "", Key: "", Source: "", Kind: TextKind, File: "",
		InputHash: "", TextHash: "", CreateTime: time.Time{},
	}, segments), nil
}
