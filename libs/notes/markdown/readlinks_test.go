package markdown

import (
	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// readLinks is the relationships written in the `links:` block of a document.
func readLinks(d *Document) ([]domain.Link, error) {
	node, err := d.readMapping()
	if err != nil || node == nil {
		return nil, err
	}
	var items *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "links" {
			items = node.Content[i+1]
			break
		}
	}
	if items == nil || items.Kind != yaml.SequenceNode {
		return nil, nil
	}
	out := make([]domain.Link, 0, len(items.Content))
	for _, item := range items.Content {
		out = append(out, linkOf(item))
	}
	return out, nil
}
