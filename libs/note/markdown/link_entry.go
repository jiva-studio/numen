package markdown

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func isOwnedKey(key string) bool {
	switch key {
	case "to", "role", "type", "note", "label", "marker":
		return true
	}
	return false
}

func renderEntry(l domain.Link, indent, eol string) ([]byte, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT reason="a top-level sequence of one mapping aligns its keys after the sequence dash regardless of SetIndent"
	enc.SetIndent(2)
	// gomutants:disable-next-line BRANCH_IF reason="encoding a linkEntry slice to bytes.Buffer cannot fail"
	if err := enc.Encode([]linkEntry{{
		To:     l.Target.GetWritten(),
		Role:   string(l.Role),
		Type:   l.Type,
		Note:   l.Why,
		Label:  l.Label,
		Marker: string(l.Marker),
	}}); err != nil {
		return nil, err
	}
	// gomutants:disable-next-line BRANCH_IF reason="closing a yaml.Encoder backed by bytes.Buffer cannot fail"
	if err := enc.Close(); err != nil {
		return nil, err
	}

	text := bytes.TrimRight(out.Bytes(), "\n")
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="slice capacity hint does not alter the rendered block bytes"
	block := make([]byte, 0, len(text)+(bytes.Count(text, []byte("\n"))+1)*(len(indent)+len(eol)))
	for {
		line, rest, more := bytes.Cut(text, []byte("\n"))
		block = append(block, indent...)
		block = append(block, line...)
		block = append(block, eol...)
		if !more {
			return block, nil
		}
		text = rest
	}
}

func linkOf(item *yaml.Node) domain.Link {
	var l domain.Link
	if item.Kind != yaml.MappingNode {
		return l
	}
	for i := 0; i+1 < len(item.Content); i += 2 {
		value := strings.TrimSpace(item.Content[i+1].Value)
		switch item.Content[i].Value {
		case "to":
			l.Target = domain.ParseAddress(value)
		case "role":
			l.Role = domain.LinkRole(value)
		case "type":
			l.Type = value
		case "note":
			l.Why = value
		case "label":
			l.Label = value
		case "marker":
			l.Marker = domain.LinkMarker(value)
		}
	}
	return l
}

func isReadable(item *yaml.Node) bool {
	// gomutants:disable-next-line BRANCH_IF reason="linkOf returns a zero Link with empty Target.Value when item is not a MappingNode"
	if item.Kind != yaml.MappingNode {
		return false
	}
	l := linkOf(item)
	return l.Target.Value != "" && domain.IsKnownRole(l.Role) && domain.IsWritableMarker(l.Marker)
}

func hasOnlyOwnedKeys(item *yaml.Node) bool {
	if item.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(item.Content); i += 2 {
		if !isOwnedKey(item.Content[i].Value) {
			return false
		}
	}
	return true
}

func keepLinkNotation(was, to string) string {
	// gomutants:disable-next-line EXPRESSION_REMOVE,INVERT_LOGICAL,INTEGER_DECREMENT reason="any string with both [[ prefix and ]] suffix has length at least 4"
	if !strings.HasPrefix(was, "[[") || !strings.HasSuffix(was, "]]") || len(was) < 4 {
		return to
	}
	return "[[" + to + getWikilinkSuffix(was[2:len(was)-2]) + "]]"
}

// linkEntry is one record of the `links:` block, in the order the format
// specification lists the fields.
type linkEntry struct {
	To     string `yaml:"to"`
	Role   string `yaml:"role"`
	Type   string `yaml:"type,omitempty"`
	Note   string `yaml:"note,omitempty"`
	Label  string `yaml:"label,omitempty"`
	Marker string `yaml:"marker,omitempty"`
}
