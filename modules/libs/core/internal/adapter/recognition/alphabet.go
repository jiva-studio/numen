package recognition

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// A recogniser carries the characters it can write inside its own file, under
// the `character` key of the model's metadata. Asking the file is the only way
// to be sure: a class count typed into a configuration that disagrees with the
// graph produces text that is wrong in a way nothing downstream can see.
const alphabetKey = "character"

// The fields of an ONNX file that this reads. A model is a protobuf message,
// and only the one field is wanted, so the rest are stepped over by their
// length.
const (
	metadataField = 14 // ModelProto.metadata_props
	keyField      = 1  // StringStringEntryProto.key
	valueField    = 2  // StringStringEntryProto.value
)

// alphabet is how many characters a recogniser knows and where they are written
// down for it.
//
// The configuration may name a file of its own, and then that file and the
// count beside it are used as given: a model exported by hand, or one whose
// alphabet somebody is changing, is a thing a person may have.
func alphabet(model string, cfg RecogniserModel) (int64, string, error) {
	if cfg.Dict != "" {
		if cfg.Classes <= 0 {
			return 0, "", fmt.Errorf("a dictionary was named without saying how many characters the model knows")
		}
		return cfg.Classes, cfg.Dict, nil
	}

	characters, err := metadata(model, alphabetKey)
	if err != nil {
		return 0, "", fmt.Errorf("%s carries no alphabet, and none was configured: %w", model, err)
	}
	lines := strings.Split(strings.TrimRight(characters, "\n"), "\n")

	// The model writes one class a character, and two more: the blank a decoder
	// collapses on, and a space. A decoder reads a class as the entry one
	// before it, so the dictionary carries every class but the blank.
	classes := int64(len(lines)) + 2
	lines = append(lines, " ")
	if cfg.Classes > 0 && cfg.Classes != classes {
		return 0, "", fmt.Errorf("%s knows %d characters and the settings say %d", model, classes, cfg.Classes)
	}

	file, err := os.CreateTemp("", "numen-alphabet-*.txt")
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		return 0, "", err
	}
	return classes, file.Name(), nil
}

// metadata is one entry of a model's metadata.
//
// Only the top level of the message is walked, and every field but the one
// wanted is stepped over by its length — including the graph, which is all of
// the weights.
func metadata(path, key string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for at := 0; at < len(raw); {
		field, wire, size, ok := tag(raw[at:])
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		at += size

		body, size, ok := payload(raw[at:], wire)
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		at += size

		if field != metadataField {
			continue
		}
		if k, v, ok := entry(body); ok && k == key {
			return v, nil
		}
	}
	return "", fmt.Errorf("no %q in the model's metadata", key)
}

// entry is one key and value of a metadata property.
func entry(raw []byte) (key, value string, ok bool) {
	for at := 0; at < len(raw); {
		field, wire, size, good := tag(raw[at:])
		if !good {
			return "", "", false
		}
		at += size
		body, size, good := payload(raw[at:], wire)
		if !good {
			return "", "", false
		}
		at += size

		switch field {
		case keyField:
			key = string(body)
		case valueField:
			value = string(body)
		}
	}
	return key, value, key != ""
}

// tag is a field's number and how it is encoded.
func tag(raw []byte) (field int, wire byte, size int, ok bool) {
	n, size := binary.Uvarint(raw)
	if size <= 0 {
		return 0, 0, 0, false
	}
	return int(n >> 3), byte(n & 7), size, true
}

// payload is one field's bytes, and how many bytes it took to say them. A field
// this does not want is stepped over the same way as one it does.
func payload(raw []byte, wire byte) (body []byte, size int, ok bool) {
	switch wire {
	case 0: // a varint
		_, n := binary.Uvarint(raw)
		if n <= 0 {
			return nil, 0, false
		}
		return nil, n, true
	case 1: // eight bytes
		if len(raw) < 8 {
			return nil, 0, false
		}
		return nil, 8, true
	case 2: // a length and then that many bytes
		length, n := binary.Uvarint(raw)
		if n <= 0 || uint64(len(raw)-n) < length {
			return nil, 0, false
		}
		return raw[n : n+int(length)], n + int(length), true
	case 5: // four bytes
		if len(raw) < 4 {
			return nil, 0, false
		}
		return nil, 4, true
	}
	return nil, 0, false
}
