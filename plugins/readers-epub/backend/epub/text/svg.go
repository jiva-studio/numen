package text

import (
	"bytes"
	"encoding/xml"
	"io"
)

// FindSVGImage returns the image an SVG wraps: the first image element in it
// naming an entry the archive holds, or one embedded in the file itself.
func FindSVGImage(base string, raw []byte, has func(string) bool) (string, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.Strict = false
	decoder.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "image" {
			continue
		}
		for _, a := range start.Attr {
			// An SVG names what it draws with href, and with xlink:href in the
			// version most books are written in.
			if a.Name.Local != "href" {
				continue
			}
			switch pos := resolveImageSource(base, a.Value); {
			case pos == "":
			case isInlineImage(pos), has(pos):
				return pos, true
			}
		}
	}
}
